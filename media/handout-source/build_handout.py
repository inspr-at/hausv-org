from pathlib import Path
import argparse
import io
from xml.etree import ElementTree
import hashlib
import json
import re
from urllib.parse import quote

from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.lib.colors import HexColor, Color
from reportlab.lib.pagesizes import A4
from reportlab.lib.utils import ImageReader
from reportlab.lib.styles import ParagraphStyle
from reportlab.platypus import Paragraph
from reportlab.graphics.barcode import qr
from reportlab.graphics.shapes import Drawing, Rect
from reportlab.graphics.svgpath import SvgPath
from reportlab.graphics import renderPDF
from pypdf import PdfReader, PdfWriter
from PIL import Image
from pypdf.generic import DictionaryObject, NameObject, ArrayObject, FloatObject, NumberObject, DecodedStreamObject, TextStringObject


ROOT = Path(__file__).resolve().parent
WORK = ROOT
ASSETS = WORK / 'assets'
parser=argparse.ArgumentParser(description='Build the one-page HAUSV customer handout.')
parser.add_argument('--lang',choices=('de','en','es'),default='de')
parser.add_argument('--output',type=Path)
args=parser.parse_args()
OUT = args.output or ROOT.parent/f'HAUSV-Professional_Handout-{args.lang}.pdf'
CONFIG=json.loads((ROOT/'build-config.json').read_text())
COPY=json.loads((ROOT/'locales.json').read_text())[args.lang]
W, H = A4
M = 35
CW = W - 2 * M

PAPER = '#F7F3EA'
INK = '#20251F'
GREEN = '#1C4032'
NAV = '#172019'
MUTED = '#60695F'
GOLD = '#8A7B3F'
GOLD_LIGHT = '#E7C574'
LINE = '#DEDCCC'
WHITE = '#FFFEFB'
PALE = '#E9EDE3'

FONT_DIR = Path('/System/Library/Fonts/Supplemental')
for name, filename in [('Body', 'Arial.ttf'), ('BodyBold', 'Arial Bold.ttf'),
                       ('Display', 'Georgia.ttf'), ('DisplayBold', 'Georgia Bold.ttf')]:
    pdfmetrics.registerFont(TTFont(name, str(FONT_DIR / filename)))
pdfmetrics.registerFontFamily('Body', normal='Body', bold='BodyBold', italic='Body', boldItalic='BodyBold')

MEASUREMENTS = []


def rect(c, x, y, w, h, fill, radius=0, stroke=None, line_width=.6):
    c.setFillColor(HexColor(fill))
    c.setLineWidth(line_width)
    if stroke:
        c.setStrokeColor(HexColor(stroke))
    if radius:
        c.roundRect(x, H-y-h, w, h, radius, fill=1, stroke=int(stroke is not None))
    else:
        c.rect(x, H-y-h, w, h, fill=1, stroke=int(stroke is not None))


def rule(c, x1, y1, x2, y2=None, color=LINE, width=.6):
    c.setStrokeColor(HexColor(color))
    c.setLineWidth(width)
    c.line(x1, H-y1, x2, H-(y1 if y2 is None else y2))


def text(c, s, x, y, size=10, font='Body', color=INK, align='left', tracking=0):
    # y is the top of the em box; keep baseline placement consistent across all roles.
    baseline = H-y-size*.8
    width = pdfmetrics.stringWidth(s, font, size) + max(0, len(s)-1)*tracking
    actual_x = x-width if align=='right' else (x-width/2 if align=='center' else x)
    c.saveState()
    obj = c.beginText(actual_x, baseline)
    obj.setFont(font, size)
    obj.setFillColor(HexColor(color))
    obj.setCharSpace(tracking)
    obj.textOut(s)
    c.drawText(obj)
    c.restoreState()
    MEASUREMENTS.append({'text':s,'x':round(actual_x,2),'y':round(y,2),'w':round(width,2),'h':size,'font':font})
    return width


def para(c, s, x, y, width, size=10, leading=14.1, color=INK, font='Body', max_height=None):
    style = ParagraphStyle('p', fontName=font, fontSize=size, leading=leading,
                           textColor=HexColor(color), spaceAfter=0, allowWidows=0,
                           allowOrphans=0, splitLongWords=False)
    p = Paragraph(s, style)
    pw, ph = p.wrap(width, H)
    if max_height and ph > max_height+.1:
        raise ValueError(f'Paragraph exceeds {max_height}: {ph}: {s}')
    p.drawOn(c,x,H-y-ph)
    MEASUREMENTS.append({'text':s,'x':x,'y':y,'w':width,'h':round(ph,2)})
    return y+ph


def svg_mark(c, path, x, y, width):
    # Preserve original path geometry and stroke from the supplied current HAUSV SVG.
    root = ElementTree.parse(path).getroot()
    vb = [float(n) for n in root.attrib['viewBox'].split()]
    scale = width/vb[2]
    c.saveState()
    c.translate(x,H-y)
    c.scale(scale,-scale)
    c.setStrokeColor(HexColor(root.attrib['stroke']))
    c.setLineWidth(float(root.attrib['stroke-width']))
    c.setLineCap(1)
    c.setLineJoin(1)
    for element in root:
        if not element.tag.endswith('path'):
            continue
        tokens = re.findall(r'[MmLlHhVvZz]|[-+]?\d*\.?\d+(?:e[-+]?\d+)?', element.attrib['d'])
        p=c.beginPath(); px=py=0; mode=None; i=0
        while i<len(tokens):
            if tokens[i].isalpha():
                mode=tokens[i]; i+=1
                if mode in 'Zz': p.close(); continue
            if mode in 'MmLl':
                a,b=float(tokens[i]),float(tokens[i+1]); i+=2
                if mode.islower(): a+=px; b+=py
                px,py=a,b
                if mode in 'Mm': p.moveTo(px,py); mode='l' if mode=='m' else 'L'
                else: p.lineTo(px,py)
            elif mode in 'Hh':
                a=float(tokens[i]); i+=1; px=a+px if mode=='h' else a; p.lineTo(px,py)
            elif mode in 'Vv':
                a=float(tokens[i]); i+=1; py=a+py if mode=='v' else a; p.lineTo(px,py)
            else: raise ValueError(f'Unexpected SVG operation {mode}')
        c.drawPath(p,stroke=1,fill=0)
    c.restoreState()


def icon(c, name, x, y, size=17, color=GREEN, line=1.2):
    # Small supporting UI-style icons; these are not brand marks.
    c.saveState(); c.translate(x,H-y-size); c.scale(size/24,size/24)
    c.setLineWidth(line*24/size); c.setLineCap(1); c.setLineJoin(1)
    c.setStrokeColor(HexColor(color)); c.setFillColor(HexColor(color))
    p=c.beginPath()
    if name=='inbox':
        p.moveTo(3,8);p.lineTo(3,19);p.lineTo(21,19);p.lineTo(21,8);p.lineTo(16,8);p.lineTo(14,5);p.lineTo(10,5);p.lineTo(8,8);p.close()
        p.moveTo(3,11);p.lineTo(8,11);p.lineTo(10,8);p.lineTo(14,8);p.lineTo(16,11);p.lineTo(21,11)
    elif name=='spark':
        p.moveTo(12,22);p.lineTo(14.5,14.5);p.lineTo(22,12);p.lineTo(14.5,9.5);p.lineTo(12,2);p.lineTo(9.5,9.5);p.lineTo(2,12);p.lineTo(9.5,14.5);p.close()
    elif name=='check':
        c.circle(12,12,9,stroke=1,fill=0);p.moveTo(7,12);p.lineTo(10.5,8.5);p.lineTo(17.5,15.5)
    elif name=='buildings':
        p.moveTo(3,3);p.lineTo(3,15);p.lineTo(10,15);p.lineTo(10,21);p.lineTo(20,21);p.lineTo(20,3);p.close()
        for a,b in [(6,11),(6,7),(13,17),(17,17),(13,13),(17,13),(13,9),(17,9)]:
            p.moveTo(a,b);p.lineTo(a+.4,b)
    elif name=='file':
        p.moveTo(5,2);p.lineTo(5,22);p.lineTo(14,22);p.lineTo(20,16);p.lineTo(20,2);p.close();p.moveTo(14,22);p.lineTo(14,16);p.lineTo(20,16)
        p.moveTo(9,11);p.lineTo(16,11);p.moveTo(9,7);p.lineTo(16,7)
    elif name=='people':
        c.circle(9,17,3,stroke=1,fill=0);c.circle(18,16,2.5,stroke=1,fill=0)
        p.moveTo(2,3);p.lineTo(2,7);p.curveTo(2,13,16,13,16,7);p.lineTo(16,3)
        p.moveTo(19,10);p.curveTo(22,10,23,8,23,6);p.lineTo(23,3)
    elif name=='shield':
        p.moveTo(12,22);p.lineTo(21,18);p.lineTo(21,11);p.curveTo(21,6,16,2,12,1);p.curveTo(8,2,3,6,3,11);p.lineTo(3,18);p.close()
        p.moveTo(7,12);p.lineTo(11,8);p.lineTo(17,15)
    elif name=='arrow':
        p.moveTo(3,12);p.lineTo(20,12);p.moveTo(14,18);p.lineTo(20,12);p.lineTo(14,6)
    c.drawPath(p,stroke=1,fill=0);c.restoreState()


def feature_icon(c, filename, x, y, size=11):
    # Repository Lucide SVGs stay vector paths, including their native arcs.
    root=ElementTree.parse(ASSETS/'icons'/filename).getroot()
    drawing=Drawing(24,24)
    common=dict(fillColor=None,strokeColor=HexColor(GREEN),strokeWidth=1.8,
                strokeLineCap=1,strokeLineJoin=1)
    for element in root:
        tag=element.tag.rsplit('}',1)[-1]
        if tag=='path':
            drawing.add(SvgPath(element.attrib['d'],**common))
        elif tag=='rect':
            attr=element.attrib
            radius=float(attr.get('rx',0))
            drawing.add(Rect(float(attr.get('x',0)),float(attr.get('y',0)),
                             float(attr['width']),float(attr['height']),
                             rx=radius,ry=float(attr.get('ry',radius)),**common))
        else:
            raise ValueError('Unsupported feature SVG element: '+tag)
    c.saveState()
    c.translate(x,H-y)
    c.scale(size/24,-size/24)
    renderPDF.draw(drawing,c,0,0)
    c.restoreState()


def qr_code(c, url, x, y, size):
    widget=qr.QrCodeWidget(url,barLevel='M',barBorder=4)
    bounds=widget.getBounds(); sx=size/(bounds[2]-bounds[0]); sy=size/(bounds[3]-bounds[1])
    drawing=Drawing(size,size,transform=[sx,0,0,sy,0,0]);drawing.add(widget)
    renderPDF.draw(drawing,c,x,H-y-size)
    c.linkURL(url,(x,H-y-size,x+size,H-y),relative=0,thickness=0)


def build():
    OUT.parent.mkdir(parents=True,exist_ok=True)
    buffer=io.BytesIO()
    c=canvas.Canvas(buffer,pagesize=A4,pageCompression=1)
    c.setTitle('HAUSV Professional | '+' '.join(COPY['headline']))
    c.setAuthor('Markus Barta · Augmentoring')
    c.setSubject(COPY['subject']+' · '+COPY['git_label']+' '+CONFIG['git_basis'])
    c.setKeywords('HAUSV, Hausverwaltung, Portfolio, KI-Posteingang, Energie, Peak Shaving, Augmentoring')
    c.setCreator('Augmentoring · Editorial PDF')
    rect(c,0,0,W,H,PAPER)

    # Masthead: the supplied HAUSV mark with the approved shared-wall correction.
    rect(c,M,28,44,33,NAV,radius=7)
    svg_mark(c,ASSETS/'hausv-mark.svg',M+1,31,42)
    text(c,'HAUSV',M+55,27,23,'DisplayBold',GREEN)
    text(c,'P R O F E S S I O N A L',M+56,53,7.4,'BodyBold',GOLD)
    text(c,COPY['masthead'][0],W-M,34,8.0,'BodyBold',MUTED,align='right',tracking=.9)
    text(c,COPY['masthead'][1],W-M,47,8.0,'BodyBold',MUTED,align='right',tracking=.9)
    rule(c,M,78,W-M)

    # Five-building portfolio derived from the user's selected image. Preserve
    # real transparency when present; never flatten an alpha-bearing cutout.
    hero_x,hero_y=265,109
    hero_w=W-M-hero_x
    hero_y-=CONFIG.get('hero_top_extension_px',0)*hero_w/1452
    photo_for_pdf=io.BytesIO()
    with Image.open(ASSETS/CONFIG.get('hero_image','hausv-portfolio.png')) as photo:
        hero_h=hero_w*photo.height/photo.width
        has_alpha='A' in photo.getbands() and photo.getchannel('A').getextrema()[0]<255
        if has_alpha:
            photo.save(photo_for_pdf,format='PNG')
        else:
            photo.convert('RGB').save(photo_for_pdf,format='JPEG',quality=95,subsampling=0,optimize=True)
    hero_box=(hero_x,H-hero_y-hero_h,hero_w,hero_h)
    hero_masks=[('GS_Hero',hero_box,6,10)]
    photo_for_pdf.seek(0)
    if not has_alpha:
        c.addLiteral('q /GS_Hero gs')
    c.drawImage(ImageReader(photo_for_pdf),hero_x,H-hero_y-hero_h,width=hero_w,height=hero_h,
                preserveAspectRatio=True,anchor='c',mask='auto')
    if not has_alpha:
        c.addLiteral('Q')
    # Separate rear properties use page-space positions, so the requested
    # centimeter offsets and the modern property's 110% scale stay deterministic.
    for index,component in enumerate(CONFIG.get('hero_components',[])):
        encoded=io.BytesIO()
        with Image.open(ASSETS/component['file']) as photo:
            photo.convert('RGB').save(encoded,format='JPEG',quality=95,subsampling=0,optimize=True)
        encoded.seek(0)
        asset=ImageReader(encoded)
        x,y,w,h=component['box_pt']
        cx,cy,cw,ch=component.get('clip_pt',component['box_pt'])
        state_name=f'GS_Component_{index}'
        hero_masks.append((state_name,(cx,H-cy-ch,cw,ch),1,1))
        c.saveState()
        clip=c.beginPath()
        clip.rect(cx,H-cy-ch,cw,ch)
        c.clipPath(clip,stroke=0,fill=0)
        c.addLiteral('/'+state_name+' gs')
        c.drawImage(asset,x,H-y-h,width=w,height=h,preserveAspectRatio=True,
                    anchor='c',mask='auto')
        c.restoreState()
    text(c,COPY['headline'][0],M,101,34,'DisplayBold',GREEN)
    text(c,COPY['headline'][1],M,144,25.5,'Display',MUTED)
    para(c,COPY['intro'],
         M,192,197,11.2,15.2,MUTED,max_height=80)
    rule(c,M,274,M+26,color=GOLD,width=1.8)
    para(c,'<b>'+'<br/>'.join(COPY['hook'])+'</b>',M,286,197,11.7,15.0,GREEN,max_height=32)

    # One coherent process instead of a catalogue of small feature cards.
    bx,by,bw,bh=M,331,CW,153
    rect(c,bx,by,bw,bh,GREEN,radius=10)
    text(c,COPY['process_title'],bx+19,by+18,18.5,'Display',WHITE)
    text(c,COPY['process_kicker'],bx+bw-19,by+23,8.2,'BodyBold',GOLD_LIGHT,align='right',tracking=1.15)
    inner= bw-38
    gap=19
    stepw=(inner-2*gap)/3
    for i,step in enumerate(COPY['steps']):
        n,title,body=f'{i+1:02d}',step['title'],step['body']
        x=bx+19+i*(stepw+gap)
        text(c,n,x,by+57,7.8,'BodyBold',GOLD_LIGHT,tracking=.4)
        step_title_width=text(c,title,x+19,by+55,10.1,'BodyBold',WHITE)
        assert step_title_width<=stepw-19,'Process title exceeds column'
        para(c,body,x,by+75,stepw,10.0,12.8,'#E7EEE8',max_height=52)
        if i<2:
            rule(c,x+stepw+gap/2,by+55,x+stepw+gap/2,by+115,'#4B695B',.5)
    rule(c,bx+19,by+126,bx+bw-19,color='#4B695B',width=.5)
    control_width=text(c,COPY['controls_label'],bx+19,by+136,7.1,'BodyBold',GOLD_LIGHT,tracking=.45)
    modes_width=text(c,COPY['controls_modes'],bx+bw-19,by+136,7.7,'Body',WHITE,align='right')
    assert control_width+modes_width+12<=inner,'Control labels overlap'

    features_title_width=text(c,COPY['features_title'],M,501,19.0,'Display',GREEN)
    assert features_title_width<=CW,'Features heading exceeds page width'
    gap=21; colw=(CW-2*gap)/3
    feature_svgs=['calendar-days.svg','file-check-2.svg','plug-zap.svg',
                  'activity.svg','gauge.svg','sliders-horizontal.svg']
    for i,feature in enumerate(COPY['features']):
        title,body=feature['title'],feature['body']
        x=M+(i%3)*(colw+gap)
        y=532+(i//3)*79
        rule(c,x,y,x+colw,color=LINE,width=.7)
        feature_icon(c,feature_svgs[i],x,y+8.8,size=11)
        title_width=text(c,title,x+15,y+10,10.3,'BodyBold',GREEN)
        assert title_width<=colw-15,'Feature title collides with next column'
        para(c,body,x,y+29,colw,10.0,12.6,MUTED,max_height=40 if i<3 else 51)

    # Trust and breadth, kept readable and free from unverified certifications.
    text(c,COPY['trust_line'],
         M,700,8.5,'Body',MUTED)

    # The call to action is intentionally a pilot conversation, not an invented price promise.
    rect(c,M,716,CW,81,WHITE,radius=7,stroke=LINE)
    text(c,COPY['cta_title'],M+13,726,18.3,'Display',GREEN)
    para(c,COPY['cta_body'],
         M+13,752,410,9.3,12.3,MUTED,max_height=15)
    email='sales@augmentoring.com'
    label=email
    tw=text(c,label,M+13,776,9.3,'BodyBold',GREEN)
    c.linkURL('mailto:'+email+'?subject='+quote(COPY['mail_subject']),
              (M+13,H-788,M+13+tw,H-774),relative=0,thickness=0)
    qr_code(c,CONFIG['demo_url'],W-M-71,719,60)
    text(c,'hausv.agm.ng',W-M-41,784,8.1,'BodyBold',GREEN,align='center')
    c.linkURL(CONFIG['demo_url'],(W-M-71,H-795,W-M-11,H-717),relative=0,thickness=0)

    rule(c,M,807,W-M,color=LINE,width=.6)
    c.drawImage(str(ASSETS/'augmentoring-monochrome.png'),M,H-823,
                width=106,height=106*156/1998,mask='auto')
    text(c,COPY['footer_service'],W/2,815,7.1,'Body',MUTED,align='center')
    footer=COPY['git_label']+' '+CONFIG['git_basis'][:7]+' · '+CONFIG['date']
    fw=text(c,footer,W-M,815,7.1,'Body',MUTED,align='right')
    c.linkURL(CONFIG['git_url'],(W-M-fw,H-827,W-M,H-812),relative=0,thickness=0)

    c.showPage();c.save()
    # A subtle PDF-native perimeter fade blends an opaque paper-colored image.
    # A genuine cutout uses its own alpha instead and has no edge fade applied.
    source=PdfReader(io.BytesIO(buffer.getvalue()))
    writer=PdfWriter()
    writer.clone_document_from_reader(source)
    writer.root_object[NameObject('/Lang')]=TextStringObject(args.lang)
    page=writer.pages[0]
    resources=page['/Resources'].get_object()
    if '/ExtGState' not in resources: resources[NameObject('/ExtGState')]=DictionaryObject()
    for state_name,(hx,hy,hw,hh),fade_x,fade_y in hero_masks:
        operations=['0 g 0 0 %.4f %.4f re f'%(W,H)]
        for i in range(181):
            t=i/180
            gray=t*t*(3-2*t)
            dx,dy=fade_x*t,fade_y*t
            operations.append('%.6f g %.4f %.4f %.4f %.4f re f'%(gray,hx+dx,hy+dy,hw-2*dx,hh-2*dy))
        mask=DecodedStreamObject()
        mask.set_data(('\n'.join(operations)).encode('ascii'))
        mask.update({NameObject('/Type'):NameObject('/XObject'),NameObject('/Subtype'):NameObject('/Form'),
                     NameObject('/FormType'):NumberObject(1),NameObject('/BBox'):ArrayObject([FloatObject(0),FloatObject(0),FloatObject(W),FloatObject(H)]),
                     NameObject('/Resources'):DictionaryObject(),
                     NameObject('/Group'):DictionaryObject({NameObject('/S'):NameObject('/Transparency'),NameObject('/CS'):NameObject('/DeviceGray')})})
        state=DictionaryObject({NameObject('/Type'):NameObject('/ExtGState'),
                                NameObject('/SMask'):DictionaryObject({NameObject('/S'):NameObject('/Luminosity'),NameObject('/G'):writer._add_object(mask)})})
        resources['/ExtGState'][NameObject('/'+state_name)]=writer._add_object(state)
    with OUT.open('wb') as f: writer.write(f)

    read=PdfReader(str(OUT))
    assert len(read.pages)==1,'PDF is not one page'
    assert all(m['x']>=0 and m['x']+m['w']<=W+.2 and m['y']+m['h']<=H for m in MEASUREMENTS), 'Text outside page'
    (WORK/f'layout-measurements-{args.lang}.json').write_text(json.dumps(MEASUREMENTS,ensure_ascii=False,indent=2))
    print(json.dumps({'pdf':str(OUT),'pages':len(read.pages),'sha256':hashlib.sha256(OUT.read_bytes()).hexdigest(),'text_characters':len(read.pages[0].extract_text())},ensure_ascii=False))


if __name__=='__main__': build()
