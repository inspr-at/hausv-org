#!/usr/bin/env python3
"""Exercise artifact enumeration and exclusive publication with a local Docker double."""
import json, os, pathlib, subprocess, tempfile, unittest
ROOT=pathlib.Path(__file__).resolve().parent.parent
class ReleaseManifestTest(unittest.TestCase):
 def test_exact_artifacts_and_no_overwrite(self):
  with tempfile.TemporaryDirectory() as td:
   temp=pathlib.Path(td)
   docker=temp/'docker'
   docker.write_text('''#!/usr/bin/env python3
import pathlib,sys
if sys.argv[1]=='create': print('fixture-container')
elif sys.argv[1]=='cp': pathlib.Path(sys.argv[3]).write_bytes(sys.argv[2].encode())
elif sys.argv[1:3]!=['container','rm']: raise SystemExit('unexpected Docker operation')
''')
   docker.chmod(0o755)
   process_env={**os.environ,'PATH':td+os.pathsep+os.environ['PATH']}
   output=temp/'release.json'
   args=['python3',str(ROOT/'scripts/release-manifest.py'),'--channel','production','--commit','a'*40,'--image','fixture-image','--image-digest','sha256:'+'b'*64,'--output',str(output)]
   first=subprocess.run(args,env=process_env,capture_output=True,text=True)
   self.assertEqual(first.returncode,0,first.stderr)
   record=json.loads(output.read_text());self.assertEqual(record['version_scheme'],'inspr-calendar-v2');self.assertEqual(len(record['artifacts']),4)
   self.assertEqual(record['artifacts'][0]['digest'],'sha256:'+'b'*64)
   self.assertEqual(len({x['coordinate'] for x in record['artifacts']}),4)
   original=output.read_bytes()
   second=subprocess.run(args,env=process_env,capture_output=True,text=True)
   self.assertNotEqual(second.returncode,0);self.assertEqual(output.read_bytes(),original)
   invalid=args.copy();invalid[invalid.index('--commit')+1]='short'
   self.assertNotEqual(subprocess.run(invalid,env=process_env,capture_output=True).returncode,0)
if __name__=='__main__':unittest.main()
