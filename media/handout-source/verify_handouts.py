"""Read-only layout/content checks for all three localized handouts."""
from pathlib import Path
import argparse
import json
import re
from urllib.parse import quote
from pypdf import PdfReader
from pypdf.generic import ContentStream

ROOT=Path(__file__).resolve().parent
parser=argparse.ArgumentParser()
parser.add_argument('--pdf-dir',type=Path,default=ROOT.parent)
args=parser.parse_args()
locales=json.loads((ROOT/'locales.json').read_text())
config=json.loads((ROOT/'build-config.json').read_text())
normalize=lambda s: re.sub(r'\s+',' ',s).strip()
reports=[]
for lang,copy in locales.items():
    path=args.pdf_dir/f'HAUSV-Professional_Handout-{lang}.pdf'
    reader=PdfReader(path)
    assert len(reader.pages)==1,(lang,'page count')
    page=reader.pages[0]
    assert abs(float(page.mediabox.width)-595.2756)<.01
    assert abs(float(page.mediabox.height)-841.8898)<.01
    assert reader.trailer['/Root']['/Lang']==lang
    extracted=normalize(page.extract_text())
    required=[]
    for key,value in copy.items():
        if key in ('subject','mail_subject'):
            continue
        if isinstance(value,str):
            required.append(value)
        else:
            for item in value:
                required.extend(item.values() if isinstance(item,dict) else [item])
    for phrase in required:
        assert normalize(phrase) in extracted,(lang,'missing or truncated',phrase)
    assert 'sales@augmentoring.com' in extracted
    assert 'markus.barta@' not in extracted
    assert '\ufffd' not in extracted
    links={(a.get_object().get('/A') or {}).get('/URI') for a in page['/Annots']}
    assert links=={config['demo_url'],config['git_url'],
                  'mailto:sales@augmentoring.com?subject='+quote(copy['mail_subject'])}
    boxes=json.loads((ROOT/f'layout-measurements-{lang}.json').read_text())
    assert all(0<=b['x'] and b['x']+b['w']<=595.48 and 0<=b['y'] and b['y']+b['h']<=841.89 for b in boxes)
    footer=next(b for b in boxes if b['text']==copy['footer_service'])
    assert abs(footer['x']+footer['w']/2-595.2756/2)<.01
    first=next(b for b in boxes if b['text']==copy['headline'][0])
    second=next(b for b in boxes if b['text']==copy['headline'][1])
    assert first['h']==34 and first['font']=='DisplayBold'
    assert second['h']==25.5 and second['font']=='Display'
    assert len([b for b in boxes if b['y'] in (542,621)])==6
    assert all(b['y']+b['h']<611 for b in boxes if b['y']==561)
    assert all(b['y']+b['h']<700 for b in boxes if b['y']==640)
    used_fonts,stack,current=set(),[],None
    ops=ContentStream(page['/Contents'],reader).operations
    for values,op in ops:
        if op==b'q': stack.append(current)
        elif op==b'Q': current=stack.pop()
        elif op==b'Tf': current=values[0]
        elif op in (b'Tj',b'TJ',b'\x27',b'\x22'): used_fonts.add(current)
    for name in used_fonts:
        font=page['/Resources']['/Font'][name].get_object()
        descriptor=font.get('/FontDescriptor')
        assert descriptor and descriptor.get_object().get('/FontFile2')
    assert sum(op==b'Do' for _,op in ops)==2+len(config.get('hero_components',[]))
    assert len(list((ROOT/'assets/icons').glob('*.svg')))==6
    reports.append({'language':lang,'pages':1,'format':'A4','copy_complete':True,
                    'links_correct':True,'fonts_embedded':True,'footer_centered':True,
                    'headline_hierarchy':True,'six_vector_icons':True,'file':str(path)})
print(json.dumps(reports,ensure_ascii=False,indent=2))
