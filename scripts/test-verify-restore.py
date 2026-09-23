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
        self.records = {'documents': [{'tenant': 'test', 'stored_filename': 'document.pdf', 'size': 9}],
                        'attachments': [], 'issues': []}

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

    def test_archive_checksum(self):
        self.records['documents'][0]['annual_statement_archive'] = {'sha256': '0' * 64}
        with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
            self.verify()


if __name__ == '__main__':
    unittest.main()
