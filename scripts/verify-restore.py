#!/usr/bin/env python3
"""Read-only verification of a local HAUSV PostgreSQL + file restore.

Run before browser QA. Prints counts/digests only, never row contents or paths.
The snapshot archive must be the independently downloaded restic dump tar.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tarfile


class VerificationError(ValueError):
    """A fixed, value-free diagnostic safe to report to the operator."""


def require(condition, message):
    if not condition:
        raise VerificationError(message)


def verify_files(data, archive, records):
    """Compare every referenced blob to metadata and the snapshot's exact bytes."""
    expected = {}
    metadata = {}
    with tarfile.open(archive) as tar:
        for member in tar:
            if not member.isfile():
                continue
            parts = Path(member.name).parts
            if data.name not in parts:
                continue
            relative = Path(*parts[parts.index(data.name) + 1:])
            require('..' not in relative.parts, 'Unsafe snapshot path')
            if len(relative.parts) == 1 and relative.suffix == '.json':
                require(str(relative) not in metadata, 'Duplicate snapshot metadata')
                metadata[str(relative)] = hashlib.sha256(tar.extractfile(member).read()).hexdigest()
            if relative.parts and relative.parts[0] in ('documents', 'attachments', 'issue-attachments', 'tenant-heroes'):
                require(str(relative) not in expected, 'Duplicate snapshot blob')
                expected[str(relative)] = hashlib.sha256(tar.extractfile(member).read()).hexdigest()
    require(expected, 'Snapshot has no blob inventory; choose the intended data snapshot')
    require('tenant_overrides.json' in metadata, 'Snapshot tenant metadata missing')
    for relative, digest in metadata.items():
        path = data / relative
        require(path.resolve().is_relative_to(data.resolve()), 'Metadata escapes data directory')
        require(path.is_file(), 'Snapshot metadata file missing')
        require(hashlib.sha256(path.read_bytes()).hexdigest() == digest, 'Metadata differs from selected snapshot')
    checked = set()

    def check(relative, size=None, digest=None):
        path = data / relative
        require(path.resolve().is_relative_to(data.resolve()), 'Blob escapes data directory')
        require(path.is_file(), 'Referenced blob missing')
        raw = path.read_bytes()
        require(size is None or len(raw) == size, 'Referenced blob size mismatch')
        actual = hashlib.sha256(raw).hexdigest()
        require(str(relative) in expected, 'Referenced blob absent from selected snapshot')
        require(actual == expected[str(relative)], 'Blob differs from selected snapshot')
        require(not digest or actual == digest, 'Archived document checksum mismatch')
        checked.add(str(relative))

    for table, folder in [('documents', 'documents'), ('attachments', 'attachments')]:
        for row in records[table]:
            if table == 'attachments' and row.get('deleted_at'):
                continue  # Tombstones retain names after the files have been removed.
            fields = [('stored_filename', 'size')]
            if table == 'attachments':
                fields += [('preview_filename', 'preview_size'), ('thumb_filename', 'thumb_size')]
            for filename, size in fields:
                if filename == 'stored_filename':
                    require(bool(row.get(filename)), 'Record has no stored filename')
                if row.get(filename):
                    require(isinstance(row.get(size), int) and row[size] > 0, 'Invalid blob size')
                    check(Path(folder) / row['tenant'] / row[filename], row[size],
                          (row.get('annual_statement_archive') or {}).get('sha256'))
    for row in records['issues']:
        for photo in row.get('photo_paths') or []:
            check(Path(photo))
    for item in records['intake_items']:
        for attachment in item.get('attachments') or []:
            require(bool(attachment.get('path')), 'Intake attachment has no path')
            require(isinstance(attachment.get('size'), int) and attachment['size'] >= 0, 'Invalid intake attachment size')
            check(Path('attachments/intake-mail') / attachment['path'], attachment['size'])
    overrides = json.loads((data / 'tenant_overrides.json').read_text())
    for row in overrides.get('tenants', {}).values():
        if row.get('hero_image'):
            check(Path('tenant-heroes') / row['hero_image'])
    # Also prove all unreferenced/legacy blobs survived extraction unchanged.
    for relative in expected:
        check(Path(relative))
    return {'snapshot_blobs': len(expected), 'verified_blobs': len(checked), 'verified_metadata_files': len(metadata)}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--container', required=True, help='Disposable local PostgreSQL container')
    p.add_argument('--data-dir', required=True, type=Path)
    p.add_argument('--snapshot-archive', required=True, type=Path)
    p.add_argument('--expected-migration', required=True)
    args = p.parse_args()

    def sql(query):
        result = subprocess.run(['docker', 'exec', '-i', args.container, 'psql', '-X', '-qAt',
                                 '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'hausv'],
                                input=query, text=True, capture_output=True)
        require(result.returncode == 0, 'Database verification query failed (details suppressed)')
        return result.stdout.strip()

    def identifier(value):
        require(re.fullmatch('[a-z_][a-z_0-9]*', value), 'Unexpected SQL identifier')
        return '"' + value + '"'

    migration = sql('SELECT max(version) FROM schema_migrations;')
    require(migration == args.expected_migration, 'Wrong snapshot/schema migration')
    require(sql("SELECT NOT rolsuper AND NOT rolbypassrls FROM pg_roles WHERE rolname='hausv_app';") == 't',
            'Application role bypasses isolation or is absent')
    require(sql("SELECT has_schema_privilege('hausv_app','public','usage') AND has_schema_privilege('hausv_app','public','create');") == 't',
            'Application schema grants missing')
    tables = sql("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename;").splitlines()
    rows = {t: int(sql('SELECT count(*) FROM ' + identifier(t))) for t in tables}
    isolated = sql("SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind='r' AND c.relname <> 'tenant' AND EXISTS (SELECT 1 FROM information_schema.columns col WHERE col.table_schema='public' AND col.table_name=c.relname AND col.column_name='tenant_id') ORDER BY c.relname;").splitlines()
    require(isolated, 'No forced row-level security')
    tenants = json.loads(sql("SELECT json_agg(tenant_id) FROM tenant;"))
    require(len(tenants) >= 2, 'Need at least two tenants to verify isolation')
    for table in isolated:
        quoted = identifier(table)
        require(sql(f"SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='public.{table}'::regclass;") == 't', 'Tenant RLS missing')
        require(sql(f'SET ROLE hausv_app; SELECT count(*) FROM {quoted};') == '0', 'Unscoped tenant data visible')
        for tenant in tenants:
            require(re.fullmatch('[0-9A-Z]{26}', tenant), 'Invalid tenant identity')
            total = sql(f"SELECT count(*) FROM {quoted} WHERE tenant_id='{tenant}';")
            visible = sql(f"SET ROLE hausv_app; SET hausv.tenant_id='{tenant}'; SELECT count(*) FROM {quoted};")
            require(total == visible, 'Tenant lane exposes wrong row count')
            require(sql(f"SET ROLE hausv_app; SET hausv.tenant_id='{tenant}'; SELECT count(*) FROM {quoted} WHERE tenant_id IS DISTINCT FROM '{tenant}';") == '0', 'Foreign tenant row visible')
    organisation_tables = sql("SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind='r' AND c.relname <> 'tenant' AND EXISTS (SELECT 1 FROM information_schema.columns col WHERE col.table_schema='public' AND col.table_name=c.relname AND col.column_name='org_key') ORDER BY c.relname;").splitlines()
    forced_count = int(sql("SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relforcerowsecurity;"))
    require(set(isolated).isdisjoint(organisation_tables) and len(isolated) + len(organisation_tables) == forced_count, 'Unverified RLS scheme')
    for table in organisation_tables:
        identifier(table)
        require(sql(f"SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='public.{table}'::regclass;") == 't', 'Organisation RLS missing')
        require(sql(f'SET ROLE hausv_app; SELECT count(*) FROM {table};') == '0', 'Unscoped organisation data visible')
        orgs = json.loads(sql(f"SELECT coalesce(json_agg(DISTINCT org_key),'[]') FROM {table};"))
        for org in orgs:
            literal = org.replace("'", "''")
            total = sql(f"SELECT count(*) FROM {table} WHERE org_key='{literal}';")
            visible = sql(f"SET ROLE hausv_app; SET app.org_key='{literal}'; SELECT count(*) FROM {table};")
            require(total == visible, 'Organisation lane exposes wrong row count')
            require(sql(f"SET ROLE hausv_app; SET app.org_key='{literal}'; SELECT count(*) FROM {table} WHERE org_key IS DISTINCT FROM '{literal}';") == '0', 'Foreign organisation row visible')
    records = {t: json.loads(sql(f"SELECT coalesce(json_agg(data::json),'[]') FROM {t};"))
               for t in ('documents', 'attachments', 'issues', 'intake_items')}
    avatars = json.loads(sql("SELECT coalesce(json_agg(json_build_array(byte_size,sha256,encode(image,'hex'))),'[]') FROM person_avatars;"))
    for size, digest, encoded in avatars:
        raw = bytes.fromhex(encoded)
        require(len(raw) == size and hashlib.sha256(raw).hexdigest() == digest, 'Avatar payload mismatch')
    imports = json.loads(sql("SELECT coalesce(json_agg(json_build_array(sha256,encode(payload,'hex'))),'[]') FROM energy_imports;"))
    for digest, encoded in imports:
        require(hashlib.sha256(bytes.fromhex(encoded)).hexdigest() == digest, 'Energy import payload mismatch')
    files = verify_files(args.data_dir, args.snapshot_archive, records)
    print(json.dumps({'migration': migration, 'tables': len(tables), 'rows': rows,
                      'forced_tenant_rls_tables': len(isolated), 'forced_organisation_rls_tables': len(organisation_tables), 'tenant_lanes': len(tenants),
                      'isolation': 'passed', 'verified_avatars': len(avatars), 'verified_energy_imports': len(imports), **files}, indent=2))


if __name__ == '__main__':
    try:
        main()
    except VerificationError as error:
        raise SystemExit('Restore verification FAILED: ' + str(error)) from None
    except Exception:
        # Exception values can include personal filenames; never print them.
        raise SystemExit('Restore verification FAILED: schema, isolation or blob check did not pass.') from None
