#!/usr/bin/env python3
"""Negative controls for the restore's independent snapshot/blob checks."""
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import sys
import unittest

sys.dont_write_bytecode = True

spec = importlib.util.spec_from_file_location('verify_restore', Path(__file__).with_name('verify-restore.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class RestoreBlobChecks(unittest.TestCase):
    def setUp(self):
        self.workspace = tempfile.TemporaryDirectory()
        self.addCleanup(self.workspace.cleanup)
        self.root = Path(self.workspace.name)
        self.data = self.root / 'snapshot-data'
        self.blob = self.data / 'documents' / 'test' / 'document.pdf'
        self.blob.parent.mkdir(parents=True)
        self.blob.write_bytes(b'%PDF-good')
        (self.data / 'tenant_overrides.json').write_text(json.dumps({'tenants': {}}))
        self.archive = self.root / 'snapshot.tar'
        with tarfile.open(self.archive, 'w') as archive:
            archive.add(self.blob, arcname='backup/snapshot-data/documents/test/document.pdf')
            archive.add(self.data / 'tenant_overrides.json', arcname='backup/snapshot-data/tenant_overrides.json')
        self.records = {'documents': [{'tenant': 'test', 'stored_filename': 'document.pdf', 'size': 9}],
                        'attachments': [], 'issues': [], 'intake_items': []}

    def verify(self):
        return module.verify_files(self.data, self.archive, self.records)

    def test_complete_restore(self):
        self.assertEqual(self.verify()['verified_blobs'], 1)

    def test_missing_file(self):
        self.blob.rename(self.root / 'held.pdf')
        with self.assertRaisesRegex(ValueError, 'missing'):
            self.verify()

    def test_same_size_corruption(self):
        self.blob.write_bytes(b'%PDF-evil')
        with self.assertRaisesRegex(ValueError, 'differs'):
            self.verify()

    def test_wrong_snapshot(self):
        with tarfile.open(self.archive, 'w') as archive:
            info = tarfile.TarInfo('backup/other-data/document.pdf')
            info.size = 9
            archive.addfile(info, io.BytesIO(b'%PDF-good'))
        with self.assertRaisesRegex(ValueError, 'no blob inventory'):
            self.verify()

    def test_path_escape(self):
        self.records['documents'][0]['stored_filename'] = '../../../../outside.pdf'
        with self.assertRaisesRegex(ValueError, 'escapes'):
            self.verify()

    def test_deleted_attachment_needs_no_file(self):
        self.records['attachments'] = [{'tenant': 'test', 'stored_filename': 'deleted.pdf',
                                        'size': 9, 'deleted_at': '2026-09-23T00:00:00Z'}]
        self.assertEqual(self.verify()['verified_blobs'], 1)

    def test_intake_reference_missing_from_both_snapshot_and_disk(self):
        self.records['intake_items'] = [{'status': 'open', 'attachments': [{'path': 'org/case/mail.pdf', 'size': 9}]}]
        with self.assertRaisesRegex(ValueError, 'missing'):
            self.verify()

    def test_intake_attachment(self):
        self.records['intake_items'] = [{'status': 'open', 'attachments': [{'path': 'org/case/mail.pdf', 'size': 9}]}]
        blob = self.data / 'attachments/intake-mail/org/case/mail.pdf'
        blob.parent.mkdir(parents=True)
        blob.write_bytes(b'%PDF-mail')
        with tarfile.open(self.archive, 'a') as archive:
            archive.add(blob, arcname='backup/snapshot-data/attachments/intake-mail/org/case/mail.pdf')
        self.assertEqual(self.verify()['verified_blobs'], 2)

    def test_empty_intake_attachment(self):
        self.records['intake_items'] = [{'attachments': [{'path': 'org/case/empty.txt', 'size': 0}]}]
        blob = self.data / 'attachments/intake-mail/org/case/empty.txt'
        blob.parent.mkdir(parents=True)
        blob.write_bytes(b'')
        with tarfile.open(self.archive, 'a') as archive:
            archive.add(blob, arcname='backup/snapshot-data/attachments/intake-mail/org/case/empty.txt')
        self.assertEqual(self.verify()['verified_blobs'], 2)

    def test_tenant_metadata_drift(self):
        (self.data / 'tenant_overrides.json').write_text('{"tenants":{"test":{}}}')
        with self.assertRaisesRegex(ValueError, 'Metadata differs'):
            self.verify()

    def test_archive_checksum(self):
        self.records['documents'][0]['annual_statement_archive'] = {'sha256': '0' * 64}
        with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
            self.verify()


if __name__ == '__main__':
    unittest.main()
