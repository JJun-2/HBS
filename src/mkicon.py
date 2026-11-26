# 아이콘(.ico) 그리기 + Go용 리소스(.syso) 만들기
import struct, io, sys
from PIL import Image, ImageDraw

def draw_icon(size=256, logo=None):
    im=Image.new('RGBA',(size,size),(0,0,0,0))
    d=ImageDraw.Draw(im)
    r=size*0.18
    d.rounded_rectangle([0,0,size-1,size-1],radius=r,fill=(22,38,61,255))
    if logo:
        lg=Image.open(logo).convert('RGBA')
        pad=int(size*0.1)
        box=size-2*pad
        k=min(box/lg.width,box/lg.height)
        lg=lg.resize((max(1,round(lg.width*k)),max(1,round(lg.height*k))),Image.LANCZOS)
        bg=Image.new('RGBA',(size,size),(0,0,0,0));bd=ImageDraw.Draw(bg)
        bd.rounded_rectangle([0,0,size-1,size-1],radius=r,fill=(255,255,255,255))
        bg.alpha_composite(lg,((size-lg.width)//2,(size-lg.height)//2))
        return bg
    # 막대그래프 + 기준선
    base=size*0.80; w=size*0.13; gap=size*0.055; x=size*0.17
    hs=[0.30,0.48,0.38,0.58]
    cols=[(51,140,237),(51,140,237),(236,70,71),(51,140,237)]
    for h,c in zip(hs,cols):
        d.rounded_rectangle([x,base-size*h,x+w,base],radius=size*0.025,fill=c+(255,))
        x+=w+gap
    y=base-size*0.52
    for i in range(int(size*0.12),int(size*0.88),int(size*0.07)):
        d.line([i,y,i+size*0.04,y],fill=(255,255,255,255),width=max(2,size//64))
    d.line([size*0.12,base+size*0.03,size*0.88,base+size*0.03],fill=(255,255,255,255),width=max(2,size//50))
    return im

def ico_images(im):
    out=[]
    for s in (16,24,32,48,64,128,256):
        b=io.BytesIO(); im.resize((s,s),Image.LANCZOS).save(b,'PNG'); out.append((s,b.getvalue()))
    return out

def write_ico(path,imgs):
    hdr=struct.pack('<HHH',0,1,len(imgs)); off=6+16*len(imgs); ents=b''; data=b''
    for s,png in imgs:
        ents+=struct.pack('<BBBBHHII',s%256,s%256,0,0,1,32,len(png),off+len(data)); data+=png
    open(path,'wb').write(hdr+ents+data)

def write_syso(path,imgs):
    # 리소스 트리: root -> [ICON(3), GROUP_ICON(14)] -> ids -> lang 0x0412 -> data entry
    grp=struct.pack('<HHH',0,1,len(imgs))
    for i,(s,png) in enumerate(imgs):
        grp+=struct.pack('<BBBBHHIH',s%256,s%256,0,0,1,32,len(png),i+1)
    types=[(3,[(i+1,png) for i,(s,png) in enumerate(imgs)]),(14,[(1,grp)])]
    def dirhdr(n): return struct.pack('<IIHHHH',0,0,0,0,0,n)
    # 크기 계산
    root_sz=16+8*len(types)
    l2_sz=[16+8*len(e) for _,e in types]
    l3_count=sum(len(e) for _,e in types)
    l3_sz=24  # 16 hdr + 1 entry
    off_l2=root_sz
    off_l3=off_l2+sum(l2_sz)
    off_de=off_l3+l3_count*l3_sz
    off_data=off_de+l3_count*16
    buf=bytearray(); relocs=[]
    # root
    buf+=dirhdr(len(types)); o=off_l2
    for (t,e),sz in zip(types,l2_sz):
        buf+=struct.pack('<II',t,0x80000000|o); o+=sz
    # level2
    k=0
    for t,e in types:
        buf+=dirhdr(len(e))
        for rid,_ in e:
            buf+=struct.pack('<II',rid,0x80000000|(off_l3+k*l3_sz)); k+=1
    # level3
    for i in range(l3_count):
        buf+=dirhdr(1)+struct.pack('<II',0x0412,off_de+i*16)
    # data entries
    blobs=[b for _,e in types for _,b in e]
    doff=off_data; positions=[]
    for b in blobs:
        positions.append(doff); doff+=len(b); doff=(doff+7)&~7
    for b,pos in zip(blobs,positions):
        relocs.append(len(buf)); buf+=struct.pack('<IIII',pos,len(b),0,0)
    assert len(buf)==off_data
    for b,pos in zip(blobs,positions):
        while len(buf)<pos: buf+=b'\0'
        buf+=b
    while len(buf)%8: buf+=b'\0'
    raw=bytes(buf)
    nrel=len(relocs)
    ptr_raw=20+40; ptr_rel=ptr_raw+len(raw); ptr_sym=ptr_rel+10*nrel
    fh=struct.pack('<HHIIIHH',0x8664,1,0,ptr_sym,1,0,0)
    sh=b'.rsrc\0\0\0'+struct.pack('<IIIIIIHHI',0,0,len(raw),ptr_raw,ptr_rel,0,nrel,0,0x40000040)
    rel=b''.join(struct.pack('<IIH',r,0,3) for r in relocs)
    sym=b'.rsrc\0\0\0'+struct.pack('<IhHBB',0,1,0,3,0)
    strtab=struct.pack('<I',4)
    open(path,'wb').write(fh+sh+raw+rel+sym+strtab)

if __name__=='__main__':
    logo=sys.argv[1] if len(sys.argv)>1 else None
    im=draw_icon(256,logo)
    imgs=ico_images(im)
    write_ico('app.ico',imgs)
    im.save('icon_preview.png')
    write_syso('rsrc_windows_amd64.syso',imgs)
    print('ok')
