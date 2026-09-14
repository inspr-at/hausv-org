#!/usr/bin/env python3
"""Freeze exact artifact coordinates after the image build, without credentials."""
import argparse, datetime, hashlib, json, pathlib, re, subprocess
p=argparse.ArgumentParser()
p.add_argument('--channel',choices=['production','demo'],required=True)
p.add_argument('--commit',required=True)
p.add_argument('--image',required=True)
p.add_argument('--image-digest',required=True)
p.add_argument('--output',required=True)
a=p.parse_args()
root=pathlib.Path(__file__).resolve().parent.parent
record=json.loads((root/'internal/version/release.json').read_text())
version=(root/'VERSION').read_text().strip()
if version!=record['version'] or not re.fullmatch(r'[1-9][0-9]{11}\.0\.0',version): raise SystemExit('invalid release coordinate')
datetime.datetime.strptime('20'+version[:12],'%Y%m%d%H%M%S')
if not re.fullmatch(r'[a-f0-9]{40}',a.commit) or not re.fullmatch(r'sha256:[a-f0-9]{64}',a.image_digest):raise SystemExit('exact source and image digest required')
container=subprocess.check_output(['docker','create',a.image],text=True).strip()
try:
 import tempfile
 with tempfile.TemporaryDirectory(prefix='hausv-release-artifacts-') as tmp:
  outputs=[('server/linux-amd64','/hausv-org'),('connector/linux-amd64','/connector-downloads/hausv-connector-linux-amd64'),('connector/linux-arm64','/connector-downloads/hausv-connector-linux-arm64')]
  artifacts=[{'coordinate':'oci/linux-amd64/'+a.channel,'image':a.image,'digest':a.image_digest,'digest_kind':'oci-manifest' if a.channel=='production' else 'docker-image-config'}]
  for coordinate,source in outputs:
   dest=pathlib.Path(tmp)/source.rsplit('/',1)[-1]
   subprocess.run(['docker','cp',container+':'+source,str(dest)],check=True,stdout=subprocess.DEVNULL)
   artifacts.append({'coordinate':coordinate+'/'+a.channel,'digest':'sha256:'+hashlib.sha256(dest.read_bytes()).hexdigest()})
finally:subprocess.run(['docker','container','rm',container],check=True,stdout=subprocess.DEVNULL)
manifest={'schema':'hausv.release-set.v1','version_scheme':record['version_scheme'],'version':version,'release_channel':a.channel,'release_sequence':record['release_sequence'],'source_commit':a.commit,'migration':record['migration'],'presentation':record['presentation'],'dependency_locks':{name:'sha256:'+hashlib.sha256((root/name).read_bytes()).hexdigest() for name in ['go.mod','go.sum','scripts/snapshot/package-lock.json']},'artifacts':artifacts}
with open(a.output,'x') as f:json.dump(manifest,f,indent=2);f.write('\n')
print('Immutable release manifest written: '+a.output)
