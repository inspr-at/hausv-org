var Qd=Object.defineProperty;var ep=(i,e,t)=>e in i?Qd(i,e,{enumerable:!0,configurable:!0,writable:!0,value:t}):i[e]=t;var Dt=(i,e,t)=>ep(i,typeof e!="symbol"?e+"":e,t);var Yi={LEFT:0,MIDDLE:1,RIGHT:2,ROTATE:0,DOLLY:1,PAN:2},Zi={ROTATE:0,PAN:1,DOLLY_PAN:2,DOLLY_ROTATE:3},Ju=0,Ll=1,$u=2;var mo=1,Qu=2,ar=3,On=0,Ft=1,Bt=2,xn=0,hs=1,Dl=2,Nl=3,Ul=4,ef=5;var ki=100,tf=101,nf=102,sf=103,rf=104,of=200,af=201,cf=202,lf=203,oa=204,aa=205,hf=206,uf=207,ff=208,df=209,pf=210,mf=211,gf=212,_f=213,xf=214,ca=0,la=1,ha=2,us=3,ua=4,fa=5,da=6,pa=7,Ol=0,yf=1,vf=2,Gn=0,Fl=1,Bl=2,kl=3,bs=4,zl=5,Hl=6,Vl=7,ml="attached",bf="detached",Gl=300,Ki=301,Ms=302,cr=303,Ha=304,go=306,Jn=1e3,Gt=1001,zi=1002,St=1003,Va=1004;var Ss=1005;var bt=1006,lr=1007;var yn=1008;var hn=1009,Wl=1010,Xl=1011,hr=1012,Ga=1013,Wn=1014,nn=1015,ni=1016,Wa=1017,Xa=1018,ur=1020,ql=35902,Yl=35899,Zl=1021,Kl=1022,vn=1023,$n=1026,ji=1027,fr=1028,qa=1029,Ji=1030,Ya=1031;var Za=1033,_o=33776,xo=33777,yo=33778,vo=33779,Ka=35840,ja=35841,Ja=35842,$a=35843,Qa=36196,ec=37492,tc=37496,nc=37488,ic=37489,bo=37490,sc=37491,rc=37808,oc=37809,ac=37810,cc=37811,lc=37812,hc=37813,uc=37814,fc=37815,dc=37816,pc=37817,mc=37818,gc=37819,_c=37820,xc=37821,yc=36492,vc=36494,bc=36495,Mc=36283,Sc=36284,Mo=36285,Tc=36286;var fs=2300,ds=2301,ra=2302,gl=2303,_l=2400,xl=2401,yl=2402,Mf=2500;var jl=0,So=1,dr=2,Sf=3200;var Ec=0,Tf=1,An="",at="srgb",Wt="srgb-linear",zr="linear",lt="srgb";var cs=7680;var vl=519,Ef=512,Af=513,wf=514,Ac=515,Rf=516,Cf=517,wc=518,Pf=519,ma=35044;var pr="300 es",Un=2e3,Zs=2001;function tp(i){for(let e=i.length-1;e>=0;--e)if(i[e]>=65535)return!0;return!1}function np(i){return ArrayBuffer.isView(i)&&!(i instanceof DataView)}function Ks(i){return document.createElementNS("http://www.w3.org/1999/xhtml",i)}function If(){let i=Ks("canvas");return i.style.display="block",i}var cu={},js=null;function Hr(...i){let e="THREE."+i.shift();js?js("log",e,...i):console.log(e,...i)}function Lf(i){let e=i[0];if(typeof e=="string"&&e.startsWith("TSL:")){let t=i[1];t&&t.isStackTrace?i[0]+=" "+t.getLocation():i[1]='Stack trace not available. Enable "THREE.Node.captureStackTrace" to capture stack traces.'}return i}function ke(...i){i=Lf(i);let e="THREE."+i.shift();if(js)js("warn",e,...i);else{let t=i[0];t&&t.isStackTrace?console.warn(t.getError(e)):console.warn(e,...i)}}function Ke(...i){i=Lf(i);let e="THREE."+i.shift();if(js)js("error",e,...i);else{let t=i[0];t&&t.isStackTrace?console.error(t.getError(e)):console.error(e,...i)}}function ls(...i){let e=i.join(" ");e in cu||(cu[e]=!0,ke(...i))}function Df(i,e,t){return new Promise(function(n,s){function r(){switch(i.clientWaitSync(e,i.SYNC_FLUSH_COMMANDS_BIT,0)){case i.WAIT_FAILED:s();break;case i.TIMEOUT_EXPIRED:setTimeout(r,t);break;default:n()}}setTimeout(r,t)})}var Nf={[ca]:la,[ha]:da,[ua]:pa,[us]:fa,[la]:ca,[da]:ha,[pa]:ua,[fa]:us},Fn=class{addEventListener(e,t){this._listeners===void 0&&(this._listeners={});let n=this._listeners;n[e]===void 0&&(n[e]=[]),n[e].indexOf(t)===-1&&n[e].push(t)}hasEventListener(e,t){let n=this._listeners;return n===void 0?!1:n[e]!==void 0&&n[e].indexOf(t)!==-1}removeEventListener(e,t){let n=this._listeners;if(n===void 0)return;let s=n[e];if(s!==void 0){let r=s.indexOf(t);r!==-1&&s.splice(r,1)}}dispatchEvent(e){let t=this._listeners;if(t===void 0)return;let n=t[e.type];if(n!==void 0){e.target=this;let s=n.slice(0);for(let r=0,o=s.length;r<o;r++)s[r].call(this,e);e.target=null}}},Yt=["00","01","02","03","04","05","06","07","08","09","0a","0b","0c","0d","0e","0f","10","11","12","13","14","15","16","17","18","19","1a","1b","1c","1d","1e","1f","20","21","22","23","24","25","26","27","28","29","2a","2b","2c","2d","2e","2f","30","31","32","33","34","35","36","37","38","39","3a","3b","3c","3d","3e","3f","40","41","42","43","44","45","46","47","48","49","4a","4b","4c","4d","4e","4f","50","51","52","53","54","55","56","57","58","59","5a","5b","5c","5d","5e","5f","60","61","62","63","64","65","66","67","68","69","6a","6b","6c","6d","6e","6f","70","71","72","73","74","75","76","77","78","79","7a","7b","7c","7d","7e","7f","80","81","82","83","84","85","86","87","88","89","8a","8b","8c","8d","8e","8f","90","91","92","93","94","95","96","97","98","99","9a","9b","9c","9d","9e","9f","a0","a1","a2","a3","a4","a5","a6","a7","a8","a9","aa","ab","ac","ad","ae","af","b0","b1","b2","b3","b4","b5","b6","b7","b8","b9","ba","bb","bc","bd","be","bf","c0","c1","c2","c3","c4","c5","c6","c7","c8","c9","ca","cb","cc","cd","ce","cf","d0","d1","d2","d3","d4","d5","d6","d7","d8","d9","da","db","dc","dd","de","df","e0","e1","e2","e3","e4","e5","e6","e7","e8","e9","ea","eb","ec","ed","ee","ef","f0","f1","f2","f3","f4","f5","f6","f7","f8","f9","fa","fb","fc","fd","fe","ff"],lu=1234567,Or=Math.PI/180,ps=180/Math.PI;function Tn(){let i=Math.random()*4294967295|0,e=Math.random()*4294967295|0,t=Math.random()*4294967295|0,n=Math.random()*4294967295|0;return(Yt[i&255]+Yt[i>>8&255]+Yt[i>>16&255]+Yt[i>>24&255]+"-"+Yt[e&255]+Yt[e>>8&255]+"-"+Yt[e>>16&15|64]+Yt[e>>24&255]+"-"+Yt[t&63|128]+Yt[t>>8&255]+"-"+Yt[t>>16&255]+Yt[t>>24&255]+Yt[n&255]+Yt[n>>8&255]+Yt[n>>16&255]+Yt[n>>24&255]).toLowerCase()}function et(i,e,t){return Math.max(e,Math.min(t,i))}function Jl(i,e){return(i%e+e)%e}function ip(i,e,t,n,s){return n+(i-e)*(s-n)/(t-e)}function sp(i,e,t){return i!==e?(t-i)/(e-i):0}function Fr(i,e,t){return(1-t)*i+t*e}function rp(i,e,t,n){return Fr(i,e,1-Math.exp(-t*n))}function op(i,e=1){return e-Math.abs(Jl(i,e*2)-e)}function ap(i,e,t){return i<=e?0:i>=t?1:(i=(i-e)/(t-e),i*i*(3-2*i))}function cp(i,e,t){return i<=e?0:i>=t?1:(i=(i-e)/(t-e),i*i*i*(i*(i*6-15)+10))}function lp(i,e){return i+Math.floor(Math.random()*(e-i+1))}function hp(i,e){return i+Math.random()*(e-i)}function up(i){return i*(.5-Math.random())}function fp(i){i!==void 0&&(lu=i);let e=lu+=1831565813;return e=Math.imul(e^e>>>15,e|1),e^=e+Math.imul(e^e>>>7,e|61),((e^e>>>14)>>>0)/4294967296}function dp(i){return i*Or}function pp(i){return i*ps}function mp(i){return(i&i-1)===0&&i!==0}function gp(i){return Math.pow(2,Math.ceil(Math.log(i)/Math.LN2))}function _p(i){return Math.pow(2,Math.floor(Math.log(i)/Math.LN2))}function xp(i,e,t,n,s){let r=Math.cos,o=Math.sin,a=r(t/2),c=o(t/2),l=r((e+n)/2),h=o((e+n)/2),u=r((e-n)/2),f=o((e-n)/2),d=r((n-e)/2),p=o((n-e)/2);switch(s){case"XYX":i.set(a*h,c*u,c*f,a*l);break;case"YZY":i.set(c*f,a*h,c*u,a*l);break;case"ZXZ":i.set(c*u,c*f,a*h,a*l);break;case"XZX":i.set(a*h,c*p,c*d,a*l);break;case"YXY":i.set(c*d,a*h,c*p,a*l);break;case"ZYZ":i.set(c*p,c*d,a*h,a*l);break;default:ke("MathUtils: .setQuaternionFromProperEuler() encountered an unknown order: "+s)}}function Nn(i,e){switch(e.constructor){case Float32Array:return i;case Uint32Array:return i/4294967295;case Uint16Array:return i/65535;case Uint8Array:return i/255;case Int32Array:return Math.max(i/2147483647,-1);case Int16Array:return Math.max(i/32767,-1);case Int8Array:return Math.max(i/127,-1);default:throw new Error("THREE.MathUtils: Invalid component type.")}}function ut(i,e){switch(e.constructor){case Float32Array:return i;case Uint32Array:return Math.round(i*4294967295);case Uint16Array:return Math.round(i*65535);case Uint8Array:return Math.round(i*255);case Int32Array:return Math.round(i*2147483647);case Int16Array:return Math.round(i*32767);case Int8Array:return Math.round(i*127);default:throw new Error("THREE.MathUtils: Invalid component type.")}}var Ts={DEG2RAD:Or,RAD2DEG:ps,generateUUID:Tn,clamp:et,euclideanModulo:Jl,mapLinear:ip,inverseLerp:sp,lerp:Fr,damp:rp,pingpong:op,smoothstep:ap,smootherstep:cp,randInt:lp,randFloat:hp,randFloatSpread:up,seededRandom:fp,degToRad:dp,radToDeg:pp,isPowerOfTwo:mp,ceilPowerOfTwo:gp,floorPowerOfTwo:_p,setQuaternionFromProperEuler:xp,normalize:ut,denormalize:Nn},ih=class ih{constructor(e=0,t=0){this.x=e,this.y=t}get width(){return this.x}set width(e){this.x=e}get height(){return this.y}set height(e){this.y=e}set(e,t){return this.x=e,this.y=t,this}setScalar(e){return this.x=e,this.y=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;default:throw new Error("THREE.Vector2: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;default:throw new Error("THREE.Vector2: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y)}copy(e){return this.x=e.x,this.y=e.y,this}add(e){return this.x+=e.x,this.y+=e.y,this}addScalar(e){return this.x+=e,this.y+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this}subScalar(e){return this.x-=e,this.y-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this}multiply(e){return this.x*=e.x,this.y*=e.y,this}multiplyScalar(e){return this.x*=e,this.y*=e,this}divide(e){return this.x/=e.x,this.y/=e.y,this}divideScalar(e){return this.multiplyScalar(1/e)}applyMatrix3(e){let t=this.x,n=this.y,s=e.elements;return this.x=s[0]*t+s[3]*n+s[6],this.y=s[1]*t+s[4]*n+s[7],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this}clamp(e,t){return this.x=et(this.x,e.x,t.x),this.y=et(this.y,e.y,t.y),this}clampScalar(e,t){return this.x=et(this.x,e,t),this.y=et(this.y,e,t),this}clampLength(e,t){let n=this.length();return this.divideScalar(n||1).multiplyScalar(et(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this}negate(){return this.x=-this.x,this.y=-this.y,this}dot(e){return this.x*e.x+this.y*e.y}cross(e){return this.x*e.y-this.y*e.x}lengthSq(){return this.x*this.x+this.y*this.y}length(){return Math.sqrt(this.x*this.x+this.y*this.y)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)}normalize(){return this.divideScalar(this.length()||1)}angle(){return Math.atan2(-this.y,-this.x)+Math.PI}angleTo(e){let t=Math.sqrt(this.lengthSq()*e.lengthSq());if(t===0)return Math.PI/2;let n=this.dot(e)/t;return Math.acos(et(n,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){let t=this.x-e.x,n=this.y-e.y;return t*t+n*n}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this}equals(e){return e.x===this.x&&e.y===this.y}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this}rotateAround(e,t){let n=Math.cos(t),s=Math.sin(t),r=this.x-e.x,o=this.y-e.y;return this.x=r*n-o*s+e.x,this.y=r*s+o*n+e.y,this}random(){return this.x=Math.random(),this.y=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y}};ih.prototype.isVector2=!0;var de=ih,Kt=class{constructor(e=0,t=0,n=0,s=1){this.isQuaternion=!0,this._x=e,this._y=t,this._z=n,this._w=s}static slerpFlat(e,t,n,s,r,o,a){let c=n[s+0],l=n[s+1],h=n[s+2],u=n[s+3],f=r[o+0],d=r[o+1],p=r[o+2],y=r[o+3];if(u!==y||c!==f||l!==d||h!==p){let m=c*f+l*d+h*p+u*y;m<0&&(f=-f,d=-d,p=-p,y=-y,m=-m);let g=1-a;if(m<.9995){let E=Math.acos(m),S=Math.sin(E);g=Math.sin(g*E)/S,a=Math.sin(a*E)/S,c=c*g+f*a,l=l*g+d*a,h=h*g+p*a,u=u*g+y*a}else{c=c*g+f*a,l=l*g+d*a,h=h*g+p*a,u=u*g+y*a;let E=1/Math.sqrt(c*c+l*l+h*h+u*u);c*=E,l*=E,h*=E,u*=E}}e[t]=c,e[t+1]=l,e[t+2]=h,e[t+3]=u}static multiplyQuaternionsFlat(e,t,n,s,r,o){let a=n[s],c=n[s+1],l=n[s+2],h=n[s+3],u=r[o],f=r[o+1],d=r[o+2],p=r[o+3];return e[t]=a*p+h*u+c*d-l*f,e[t+1]=c*p+h*f+l*u-a*d,e[t+2]=l*p+h*d+a*f-c*u,e[t+3]=h*p-a*u-c*f-l*d,e}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get w(){return this._w}set w(e){this._w=e,this._onChangeCallback()}set(e,t,n,s){return this._x=e,this._y=t,this._z=n,this._w=s,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._w)}copy(e){return this._x=e.x,this._y=e.y,this._z=e.z,this._w=e.w,this._onChangeCallback(),this}setFromEuler(e,t=!0){let n=e._x,s=e._y,r=e._z,o=e._order,a=Math.cos,c=Math.sin,l=a(n/2),h=a(s/2),u=a(r/2),f=c(n/2),d=c(s/2),p=c(r/2);switch(o){case"XYZ":this._x=f*h*u+l*d*p,this._y=l*d*u-f*h*p,this._z=l*h*p+f*d*u,this._w=l*h*u-f*d*p;break;case"YXZ":this._x=f*h*u+l*d*p,this._y=l*d*u-f*h*p,this._z=l*h*p-f*d*u,this._w=l*h*u+f*d*p;break;case"ZXY":this._x=f*h*u-l*d*p,this._y=l*d*u+f*h*p,this._z=l*h*p+f*d*u,this._w=l*h*u-f*d*p;break;case"ZYX":this._x=f*h*u-l*d*p,this._y=l*d*u+f*h*p,this._z=l*h*p-f*d*u,this._w=l*h*u+f*d*p;break;case"YZX":this._x=f*h*u+l*d*p,this._y=l*d*u+f*h*p,this._z=l*h*p-f*d*u,this._w=l*h*u-f*d*p;break;case"XZY":this._x=f*h*u-l*d*p,this._y=l*d*u-f*h*p,this._z=l*h*p+f*d*u,this._w=l*h*u+f*d*p;break;default:ke("Quaternion: .setFromEuler() encountered an unknown order: "+o)}return t===!0&&this._onChangeCallback(),this}setFromAxisAngle(e,t){let n=t/2,s=Math.sin(n);return this._x=e.x*s,this._y=e.y*s,this._z=e.z*s,this._w=Math.cos(n),this._onChangeCallback(),this}setFromRotationMatrix(e){let t=e.elements,n=t[0],s=t[4],r=t[8],o=t[1],a=t[5],c=t[9],l=t[2],h=t[6],u=t[10],f=n+a+u;if(f>0){let d=.5/Math.sqrt(f+1);this._w=.25/d,this._x=(h-c)*d,this._y=(r-l)*d,this._z=(o-s)*d}else if(n>a&&n>u){let d=2*Math.sqrt(1+n-a-u);this._w=(h-c)/d,this._x=.25*d,this._y=(s+o)/d,this._z=(r+l)/d}else if(a>u){let d=2*Math.sqrt(1+a-n-u);this._w=(r-l)/d,this._x=(s+o)/d,this._y=.25*d,this._z=(c+h)/d}else{let d=2*Math.sqrt(1+u-n-a);this._w=(o-s)/d,this._x=(r+l)/d,this._y=(c+h)/d,this._z=.25*d}return this._onChangeCallback(),this}setFromUnitVectors(e,t){let n=e.dot(t)+1;return n<1e-8?(n=0,Math.abs(e.x)>Math.abs(e.z)?(this._x=-e.y,this._y=e.x,this._z=0,this._w=n):(this._x=0,this._y=-e.z,this._z=e.y,this._w=n)):(this._x=e.y*t.z-e.z*t.y,this._y=e.z*t.x-e.x*t.z,this._z=e.x*t.y-e.y*t.x,this._w=n),this.normalize()}angleTo(e){return 2*Math.acos(Math.abs(et(this.dot(e),-1,1)))}rotateTowards(e,t){let n=this.angleTo(e);if(n===0)return this;let s=Math.min(1,t/n);return this.slerp(e,s),this}identity(){return this.set(0,0,0,1)}invert(){return this.conjugate()}conjugate(){return this._x*=-1,this._y*=-1,this._z*=-1,this._onChangeCallback(),this}dot(e){return this._x*e._x+this._y*e._y+this._z*e._z+this._w*e._w}lengthSq(){return this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w}length(){return Math.sqrt(this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w)}normalize(){let e=this.length();return e===0?(this._x=0,this._y=0,this._z=0,this._w=1):(e=1/e,this._x=this._x*e,this._y=this._y*e,this._z=this._z*e,this._w=this._w*e),this._onChangeCallback(),this}multiply(e){return this.multiplyQuaternions(this,e)}premultiply(e){return this.multiplyQuaternions(e,this)}multiplyQuaternions(e,t){let n=e._x,s=e._y,r=e._z,o=e._w,a=t._x,c=t._y,l=t._z,h=t._w;return this._x=n*h+o*a+s*l-r*c,this._y=s*h+o*c+r*a-n*l,this._z=r*h+o*l+n*c-s*a,this._w=o*h-n*a-s*c-r*l,this._onChangeCallback(),this}slerp(e,t){let n=e._x,s=e._y,r=e._z,o=e._w,a=this.dot(e);a<0&&(n=-n,s=-s,r=-r,o=-o,a=-a);let c=1-t;if(a<.9995){let l=Math.acos(a),h=Math.sin(l);c=Math.sin(c*l)/h,t=Math.sin(t*l)/h,this._x=this._x*c+n*t,this._y=this._y*c+s*t,this._z=this._z*c+r*t,this._w=this._w*c+o*t,this._onChangeCallback()}else this._x=this._x*c+n*t,this._y=this._y*c+s*t,this._z=this._z*c+r*t,this._w=this._w*c+o*t,this.normalize();return this}slerpQuaternions(e,t,n){return this.copy(e).slerp(t,n)}random(){let e=2*Math.PI*Math.random(),t=2*Math.PI*Math.random(),n=Math.random(),s=Math.sqrt(1-n),r=Math.sqrt(n);return this.set(s*Math.sin(e),s*Math.cos(e),r*Math.sin(t),r*Math.cos(t))}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._w===this._w}fromArray(e,t=0){return this._x=e[t],this._y=e[t+1],this._z=e[t+2],this._w=e[t+3],this._onChangeCallback(),this}toArray(e=[],t=0){return e[t]=this._x,e[t+1]=this._y,e[t+2]=this._z,e[t+3]=this._w,e}fromBufferAttribute(e,t){return this._x=e.getX(t),this._y=e.getY(t),this._z=e.getZ(t),this._w=e.getW(t),this._onChangeCallback(),this}toJSON(){return this.toArray()}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._w}},sh=class sh{constructor(e=0,t=0,n=0){this.x=e,this.y=t,this.z=n}set(e,t,n){return n===void 0&&(n=this.z),this.x=e,this.y=t,this.z=n,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;case 2:this.z=t;break;default:throw new Error("THREE.Vector3: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;default:throw new Error("THREE.Vector3: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this.z=e.z+t.z,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this.z+=e.z*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this.z=e.z-t.z,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this}multiplyVectors(e,t){return this.x=e.x*t.x,this.y=e.y*t.y,this.z=e.z*t.z,this}applyEuler(e){return this.applyQuaternion(hu.setFromEuler(e))}applyAxisAngle(e,t){return this.applyQuaternion(hu.setFromAxisAngle(e,t))}applyMatrix3(e){let t=this.x,n=this.y,s=this.z,r=e.elements;return this.x=r[0]*t+r[3]*n+r[6]*s,this.y=r[1]*t+r[4]*n+r[7]*s,this.z=r[2]*t+r[5]*n+r[8]*s,this}applyNormalMatrix(e){return this.applyMatrix3(e).normalize()}applyMatrix4(e){let t=this.x,n=this.y,s=this.z,r=e.elements,o=1/(r[3]*t+r[7]*n+r[11]*s+r[15]);return this.x=(r[0]*t+r[4]*n+r[8]*s+r[12])*o,this.y=(r[1]*t+r[5]*n+r[9]*s+r[13])*o,this.z=(r[2]*t+r[6]*n+r[10]*s+r[14])*o,this}applyQuaternion(e){let t=this.x,n=this.y,s=this.z,r=e.x,o=e.y,a=e.z,c=e.w,l=2*(o*s-a*n),h=2*(a*t-r*s),u=2*(r*n-o*t);return this.x=t+c*l+o*u-a*h,this.y=n+c*h+a*l-r*u,this.z=s+c*u+r*h-o*l,this}project(e){return this.applyMatrix4(e.matrixWorldInverse).applyMatrix4(e.projectionMatrix)}unproject(e){return this.applyMatrix4(e.projectionMatrixInverse).applyMatrix4(e.matrixWorld)}transformDirection(e){let t=this.x,n=this.y,s=this.z,r=e.elements;return this.x=r[0]*t+r[4]*n+r[8]*s,this.y=r[1]*t+r[5]*n+r[9]*s,this.z=r[2]*t+r[6]*n+r[10]*s,this.normalize()}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this}divideScalar(e){return this.multiplyScalar(1/e)}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this}clamp(e,t){return this.x=et(this.x,e.x,t.x),this.y=et(this.y,e.y,t.y),this.z=et(this.z,e.z,t.z),this}clampScalar(e,t){return this.x=et(this.x,e,t),this.y=et(this.y,e,t),this.z=et(this.z,e,t),this}clampLength(e,t){let n=this.length();return this.divideScalar(n||1).multiplyScalar(et(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this.z+=(e.z-this.z)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this.z=e.z+(t.z-e.z)*n,this}cross(e){return this.crossVectors(this,e)}crossVectors(e,t){let n=e.x,s=e.y,r=e.z,o=t.x,a=t.y,c=t.z;return this.x=s*c-r*a,this.y=r*o-n*c,this.z=n*a-s*o,this}projectOnVector(e){let t=e.lengthSq();if(t===0)return this.set(0,0,0);let n=e.dot(this)/t;return this.copy(e).multiplyScalar(n)}projectOnPlane(e){return zc.copy(this).projectOnVector(e),this.sub(zc)}reflect(e){return this.sub(zc.copy(e).multiplyScalar(2*this.dot(e)))}angleTo(e){let t=Math.sqrt(this.lengthSq()*e.lengthSq());if(t===0)return Math.PI/2;let n=this.dot(e)/t;return Math.acos(et(n,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){let t=this.x-e.x,n=this.y-e.y,s=this.z-e.z;return t*t+n*n+s*s}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)+Math.abs(this.z-e.z)}setFromSpherical(e){return this.setFromSphericalCoords(e.radius,e.phi,e.theta)}setFromSphericalCoords(e,t,n){let s=Math.sin(t)*e;return this.x=s*Math.sin(n),this.y=Math.cos(t)*e,this.z=s*Math.cos(n),this}setFromCylindrical(e){return this.setFromCylindricalCoords(e.radius,e.theta,e.y)}setFromCylindricalCoords(e,t,n){return this.x=e*Math.sin(t),this.y=n,this.z=e*Math.cos(t),this}setFromMatrixPosition(e){let t=e.elements;return this.x=t[12],this.y=t[13],this.z=t[14],this}setFromMatrixScale(e){let t=this.setFromMatrixColumn(e,0).length(),n=this.setFromMatrixColumn(e,1).length(),s=this.setFromMatrixColumn(e,2).length();return this.x=t,this.y=n,this.z=s,this}setFromMatrixColumn(e,t){return this.fromArray(e.elements,t*4)}setFromMatrix3Column(e,t){return this.fromArray(e.elements,t*3)}setFromEuler(e){return this.x=e._x,this.y=e._y,this.z=e._z,this}setFromColor(e){return this.x=e.r,this.y=e.g,this.z=e.b,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this.z=e[t+2],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e[t+2]=this.z,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this.z=e.getZ(t),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this}randomDirection(){let e=Math.random()*Math.PI*2,t=Math.random()*2-1,n=Math.sqrt(1-t*t);return this.x=n*Math.cos(e),this.y=t,this.z=n*Math.sin(e),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z}};sh.prototype.isVector3=!0;var B=sh,zc=new B,hu=new Kt,rh=class rh{constructor(e,t,n,s,r,o,a,c,l){this.elements=[1,0,0,0,1,0,0,0,1],e!==void 0&&this.set(e,t,n,s,r,o,a,c,l)}set(e,t,n,s,r,o,a,c,l){let h=this.elements;return h[0]=e,h[1]=s,h[2]=a,h[3]=t,h[4]=r,h[5]=c,h[6]=n,h[7]=o,h[8]=l,this}identity(){return this.set(1,0,0,0,1,0,0,0,1),this}copy(e){let t=this.elements,n=e.elements;return t[0]=n[0],t[1]=n[1],t[2]=n[2],t[3]=n[3],t[4]=n[4],t[5]=n[5],t[6]=n[6],t[7]=n[7],t[8]=n[8],this}extractBasis(e,t,n){return e.setFromMatrix3Column(this,0),t.setFromMatrix3Column(this,1),n.setFromMatrix3Column(this,2),this}setFromMatrix4(e){let t=e.elements;return this.set(t[0],t[4],t[8],t[1],t[5],t[9],t[2],t[6],t[10]),this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,t){let n=e.elements,s=t.elements,r=this.elements,o=n[0],a=n[3],c=n[6],l=n[1],h=n[4],u=n[7],f=n[2],d=n[5],p=n[8],y=s[0],m=s[3],g=s[6],E=s[1],S=s[4],x=s[7],A=s[2],w=s[5],I=s[8];return r[0]=o*y+a*E+c*A,r[3]=o*m+a*S+c*w,r[6]=o*g+a*x+c*I,r[1]=l*y+h*E+u*A,r[4]=l*m+h*S+u*w,r[7]=l*g+h*x+u*I,r[2]=f*y+d*E+p*A,r[5]=f*m+d*S+p*w,r[8]=f*g+d*x+p*I,this}multiplyScalar(e){let t=this.elements;return t[0]*=e,t[3]*=e,t[6]*=e,t[1]*=e,t[4]*=e,t[7]*=e,t[2]*=e,t[5]*=e,t[8]*=e,this}determinant(){let e=this.elements,t=e[0],n=e[1],s=e[2],r=e[3],o=e[4],a=e[5],c=e[6],l=e[7],h=e[8];return t*o*h-t*a*l-n*r*h+n*a*c+s*r*l-s*o*c}invert(){let e=this.elements,t=e[0],n=e[1],s=e[2],r=e[3],o=e[4],a=e[5],c=e[6],l=e[7],h=e[8],u=h*o-a*l,f=a*c-h*r,d=l*r-o*c,p=t*u+n*f+s*d;if(p===0)return this.set(0,0,0,0,0,0,0,0,0);let y=1/p;return e[0]=u*y,e[1]=(s*l-h*n)*y,e[2]=(a*n-s*o)*y,e[3]=f*y,e[4]=(h*t-s*c)*y,e[5]=(s*r-a*t)*y,e[6]=d*y,e[7]=(n*c-l*t)*y,e[8]=(o*t-n*r)*y,this}transpose(){let e,t=this.elements;return e=t[1],t[1]=t[3],t[3]=e,e=t[2],t[2]=t[6],t[6]=e,e=t[5],t[5]=t[7],t[7]=e,this}getNormalMatrix(e){return this.setFromMatrix4(e).invert().transpose()}transposeIntoArray(e){let t=this.elements;return e[0]=t[0],e[1]=t[3],e[2]=t[6],e[3]=t[1],e[4]=t[4],e[5]=t[7],e[6]=t[2],e[7]=t[5],e[8]=t[8],this}setUvTransform(e,t,n,s,r,o,a){let c=Math.cos(r),l=Math.sin(r);return this.set(n*c,n*l,-n*(c*o+l*a)+o+e,-s*l,s*c,-s*(-l*o+c*a)+a+t,0,0,1),this}scale(e,t){return ls("Matrix3: .scale() is deprecated. Use .makeScale() instead."),this.premultiply(Hc.makeScale(e,t)),this}rotate(e){return ls("Matrix3: .rotate() is deprecated. Use .makeRotation() instead."),this.premultiply(Hc.makeRotation(-e)),this}translate(e,t){return ls("Matrix3: .translate() is deprecated. Use .makeTranslation() instead."),this.premultiply(Hc.makeTranslation(e,t)),this}makeTranslation(e,t){return e.isVector2?this.set(1,0,e.x,0,1,e.y,0,0,1):this.set(1,0,e,0,1,t,0,0,1),this}makeRotation(e){let t=Math.cos(e),n=Math.sin(e);return this.set(t,-n,0,n,t,0,0,0,1),this}makeScale(e,t){return this.set(e,0,0,0,t,0,0,0,1),this}equals(e){let t=this.elements,n=e.elements;for(let s=0;s<9;s++)if(t[s]!==n[s])return!1;return!0}fromArray(e,t=0){for(let n=0;n<9;n++)this.elements[n]=e[n+t];return this}toArray(e=[],t=0){let n=this.elements;return e[t]=n[0],e[t+1]=n[1],e[t+2]=n[2],e[t+3]=n[3],e[t+4]=n[4],e[t+5]=n[5],e[t+6]=n[6],e[t+7]=n[7],e[t+8]=n[8],e}clone(){return new this.constructor().fromArray(this.elements)}};rh.prototype.isMatrix3=!0;var Ze=rh,Hc=new Ze,uu=new Ze().set(.4123908,.3575843,.1804808,.212639,.7151687,.0721923,.0193308,.1191948,.9505322),fu=new Ze().set(3.2409699,-1.5373832,-.4986108,-.9692436,1.8759675,.0415551,.0556301,-.203977,1.0569715);function yp(){let i={enabled:!0,workingColorSpace:Wt,spaces:{},convert:function(s,r,o){return this.enabled===!1||r===o||!r||!o||(this.spaces[r].transfer===lt&&(s.r=mi(s.r),s.g=mi(s.g),s.b=mi(s.b)),this.spaces[r].primaries!==this.spaces[o].primaries&&(s.applyMatrix3(this.spaces[r].toXYZ),s.applyMatrix3(this.spaces[o].fromXYZ)),this.spaces[o].transfer===lt&&(s.r=Ys(s.r),s.g=Ys(s.g),s.b=Ys(s.b))),s},workingToColorSpace:function(s,r){return this.convert(s,this.workingColorSpace,r)},colorSpaceToWorking:function(s,r){return this.convert(s,r,this.workingColorSpace)},getPrimaries:function(s){return this.spaces[s].primaries},getTransfer:function(s){return s===An?zr:this.spaces[s].transfer},getToneMappingMode:function(s){return this.spaces[s].outputColorSpaceConfig.toneMappingMode||"standard"},getLuminanceCoefficients:function(s,r=this.workingColorSpace){return s.fromArray(this.spaces[r].luminanceCoefficients)},define:function(s){Object.assign(this.spaces,s)},_getMatrix:function(s,r,o){return s.copy(this.spaces[r].toXYZ).multiply(this.spaces[o].fromXYZ)},_getDrawingBufferColorSpace:function(s){return this.spaces[s].outputColorSpaceConfig.drawingBufferColorSpace},_getUnpackColorSpace:function(s=this.workingColorSpace){return this.spaces[s].workingColorSpaceConfig.unpackColorSpace},fromWorkingColorSpace:function(s,r){return ls("ColorManagement: .fromWorkingColorSpace() has been renamed to .workingToColorSpace()."),i.workingToColorSpace(s,r)},toWorkingColorSpace:function(s,r){return ls("ColorManagement: .toWorkingColorSpace() has been renamed to .colorSpaceToWorking()."),i.colorSpaceToWorking(s,r)}},e=[.64,.33,.3,.6,.15,.06],t=[.2126,.7152,.0722],n=[.3127,.329];return i.define({[Wt]:{primaries:e,whitePoint:n,transfer:zr,toXYZ:uu,fromXYZ:fu,luminanceCoefficients:t,workingColorSpaceConfig:{unpackColorSpace:at},outputColorSpaceConfig:{drawingBufferColorSpace:at}},[at]:{primaries:e,whitePoint:n,transfer:lt,toXYZ:uu,fromXYZ:fu,luminanceCoefficients:t,outputColorSpaceConfig:{drawingBufferColorSpace:at}}}),i}var Qe=yp();function mi(i){return i<.04045?i*.0773993808:Math.pow(i*.9478672986+.0521327014,2.4)}function Ys(i){return i<.0031308?i*12.92:1.055*Math.pow(i,.41666)-.055}var Ds,ga=class{static getDataURL(e,t="image/png"){if(/^data:/i.test(e.src)||typeof HTMLCanvasElement>"u")return e.src;let n;if(e instanceof HTMLCanvasElement)n=e;else{Ds===void 0&&(Ds=Ks("canvas")),Ds.width=e.width,Ds.height=e.height;let s=Ds.getContext("2d");e instanceof ImageData?s.putImageData(e,0,0):s.drawImage(e,0,0,e.width,e.height),n=Ds}return n.toDataURL(t)}static sRGBToLinear(e){if(typeof HTMLImageElement<"u"&&e instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&e instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&e instanceof ImageBitmap){let t=Ks("canvas");t.width=e.width,t.height=e.height;let n=t.getContext("2d");n.drawImage(e,0,0,e.width,e.height);let s=n.getImageData(0,0,e.width,e.height),r=s.data;for(let o=0;o<r.length;o++)r[o]=mi(r[o]/255)*255;return n.putImageData(s,0,0),t}else if(e.data){let t=e.data.slice(0);for(let n=0;n<t.length;n++)t instanceof Uint8Array||t instanceof Uint8ClampedArray?t[n]=Math.floor(mi(t[n]/255)*255):t[n]=mi(t[n]);return{data:t,width:e.width,height:e.height}}else return ke("ImageUtils.sRGBToLinear(): Unsupported image type. No color space conversion applied."),e}},vp=0,Js=class{constructor(e=null){this.isSource=!0,Object.defineProperty(this,"id",{value:vp++}),this.uuid=Tn(),this.data=e,this.dataReady=!0,this.version=0}getSize(e){let t=this.data;return typeof HTMLVideoElement<"u"&&t instanceof HTMLVideoElement?e.set(t.videoWidth,t.videoHeight,0):typeof VideoFrame<"u"&&t instanceof VideoFrame?e.set(t.displayWidth,t.displayHeight,0):t!==null?e.set(t.width,t.height,t.depth||0):e.set(0,0,0),e}set needsUpdate(e){e===!0&&this.version++}toJSON(e){let t=e===void 0||typeof e=="string";if(!t&&e.images[this.uuid]!==void 0)return e.images[this.uuid];let n={uuid:this.uuid,url:""},s=this.data;if(s!==null){let r;if(Array.isArray(s)){r=[];for(let o=0,a=s.length;o<a;o++)s[o].isDataTexture?r.push(Vc(s[o].image)):r.push(Vc(s[o]))}else r=Vc(s);n.url=r}return t||(e.images[this.uuid]=n),n}};function Vc(i){return typeof HTMLImageElement<"u"&&i instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&i instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&i instanceof ImageBitmap?ga.getDataURL(i):i.data?{data:Array.from(i.data),width:i.width,height:i.height,type:i.data.constructor.name}:(ke("Texture: Unable to serialize Texture."),{})}var bp=0,Gc=new B,Tt=class i extends Fn{constructor(e=i.DEFAULT_IMAGE,t=i.DEFAULT_MAPPING,n=Gt,s=Gt,r=bt,o=yn,a=vn,c=hn,l=i.DEFAULT_ANISOTROPY,h=An){super(),this.isTexture=!0,Object.defineProperty(this,"id",{value:bp++}),this.uuid=Tn(),this.name="",this.source=new Js(e),this.mipmaps=[],this.mapping=t,this.channel=0,this.wrapS=n,this.wrapT=s,this.magFilter=r,this.minFilter=o,this.anisotropy=l,this.format=a,this.internalFormat=null,this.type=c,this.offset=new de(0,0),this.repeat=new de(1,1),this.center=new de(0,0),this.rotation=0,this.matrixAutoUpdate=!0,this.matrix=new Ze,this.generateMipmaps=!0,this.premultiplyAlpha=!1,this.flipY=!0,this.unpackAlignment=4,this.colorSpace=h,this.userData={},this.updateRanges=[],this.version=0,this.onUpdate=null,this.renderTarget=null,this.isRenderTargetTexture=!1,this.isArrayTexture=!!(e&&e.depth&&e.depth>1),this.pmremVersion=0,this.normalized=!1}get width(){return this.source.getSize(Gc).x}get height(){return this.source.getSize(Gc).y}get depth(){return this.source.getSize(Gc).z}get image(){return this.source.data}set image(e){this.source.data=e}updateMatrix(){this.matrix.setUvTransform(this.offset.x,this.offset.y,this.repeat.x,this.repeat.y,this.rotation,this.center.x,this.center.y)}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}clone(){return new this.constructor().copy(this)}copy(e){return this.name=e.name,this.source=e.source,this.mipmaps=e.mipmaps.slice(0),this.mapping=e.mapping,this.channel=e.channel,this.wrapS=e.wrapS,this.wrapT=e.wrapT,this.magFilter=e.magFilter,this.minFilter=e.minFilter,this.anisotropy=e.anisotropy,this.format=e.format,this.internalFormat=e.internalFormat,this.type=e.type,this.normalized=e.normalized,this.offset.copy(e.offset),this.repeat.copy(e.repeat),this.center.copy(e.center),this.rotation=e.rotation,this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrix.copy(e.matrix),this.generateMipmaps=e.generateMipmaps,this.premultiplyAlpha=e.premultiplyAlpha,this.flipY=e.flipY,this.unpackAlignment=e.unpackAlignment,this.colorSpace=e.colorSpace,this.renderTarget=e.renderTarget,this.isRenderTargetTexture=e.isRenderTargetTexture,this.isArrayTexture=e.isArrayTexture,this.userData=JSON.parse(JSON.stringify(e.userData)),this.needsUpdate=!0,this}setValues(e){for(let t in e){let n=e[t];if(n===void 0){ke(`Texture.setValues(): parameter '${t}' has value of undefined.`);continue}let s=this[t];if(s===void 0){ke(`Texture.setValues(): property '${t}' does not exist.`);continue}s&&n&&s.isVector2&&n.isVector2||s&&n&&s.isVector3&&n.isVector3||s&&n&&s.isMatrix3&&n.isMatrix3?s.copy(n):this[t]=n}}toJSON(e){let t=e===void 0||typeof e=="string";if(!t&&e.textures[this.uuid]!==void 0)return e.textures[this.uuid];let n={metadata:{version:4.7,type:"Texture",generator:"Texture.toJSON"},uuid:this.uuid,name:this.name,image:this.source.toJSON(e).uuid,mapping:this.mapping,channel:this.channel,repeat:[this.repeat.x,this.repeat.y],offset:[this.offset.x,this.offset.y],center:[this.center.x,this.center.y],rotation:this.rotation,wrap:[this.wrapS,this.wrapT],format:this.format,internalFormat:this.internalFormat,type:this.type,normalized:this.normalized,colorSpace:this.colorSpace,minFilter:this.minFilter,magFilter:this.magFilter,anisotropy:this.anisotropy,flipY:this.flipY,generateMipmaps:this.generateMipmaps,premultiplyAlpha:this.premultiplyAlpha,unpackAlignment:this.unpackAlignment};return Object.keys(this.userData).length>0&&(n.userData=this.userData),t||(e.textures[this.uuid]=n),n}dispose(){this.dispatchEvent({type:"dispose"})}transformUv(e){if(this.mapping!==Gl)return e;if(e.applyMatrix3(this.matrix),e.x<0||e.x>1)switch(this.wrapS){case Jn:e.x=e.x-Math.floor(e.x);break;case Gt:e.x=e.x<0?0:1;break;case zi:Math.abs(Math.floor(e.x)%2)===1?e.x=Math.ceil(e.x)-e.x:e.x=e.x-Math.floor(e.x);break}if(e.y<0||e.y>1)switch(this.wrapT){case Jn:e.y=e.y-Math.floor(e.y);break;case Gt:e.y=e.y<0?0:1;break;case zi:Math.abs(Math.floor(e.y)%2)===1?e.y=Math.ceil(e.y)-e.y:e.y=e.y-Math.floor(e.y);break}return this.flipY&&(e.y=1-e.y),e}set needsUpdate(e){e===!0&&(this.version++,this.source.needsUpdate=!0)}set needsPMREMUpdate(e){e===!0&&this.pmremVersion++}};Tt.DEFAULT_IMAGE=null;Tt.DEFAULT_MAPPING=Gl;Tt.DEFAULT_ANISOTROPY=1;var oh=class oh{constructor(e=0,t=0,n=0,s=1){this.x=e,this.y=t,this.z=n,this.w=s}get width(){return this.z}set width(e){this.z=e}get height(){return this.w}set height(e){this.w=e}set(e,t,n,s){return this.x=e,this.y=t,this.z=n,this.w=s,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this.w=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setW(e){return this.w=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;case 2:this.z=t;break;case 3:this.w=t;break;default:throw new Error("THREE.Vector4: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;case 3:return this.w;default:throw new Error("THREE.Vector4: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z,this.w)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this.w=e.w!==void 0?e.w:1,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this.w+=e.w,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this.w+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this.z=e.z+t.z,this.w=e.w+t.w,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this.z+=e.z*t,this.w+=e.w*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this.w-=e.w,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this.w-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this.z=e.z-t.z,this.w=e.w-t.w,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this.w*=e.w,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this.w*=e,this}applyMatrix4(e){let t=this.x,n=this.y,s=this.z,r=this.w,o=e.elements;return this.x=o[0]*t+o[4]*n+o[8]*s+o[12]*r,this.y=o[1]*t+o[5]*n+o[9]*s+o[13]*r,this.z=o[2]*t+o[6]*n+o[10]*s+o[14]*r,this.w=o[3]*t+o[7]*n+o[11]*s+o[15]*r,this}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this.w/=e.w,this}divideScalar(e){return this.multiplyScalar(1/e)}setAxisAngleFromQuaternion(e){this.w=2*Math.acos(e.w);let t=Math.sqrt(1-e.w*e.w);return t<1e-4?(this.x=1,this.y=0,this.z=0):(this.x=e.x/t,this.y=e.y/t,this.z=e.z/t),this}setAxisAngleFromRotationMatrix(e){let t,n,s,r,c=e.elements,l=c[0],h=c[4],u=c[8],f=c[1],d=c[5],p=c[9],y=c[2],m=c[6],g=c[10];if(Math.abs(h-f)<.01&&Math.abs(u-y)<.01&&Math.abs(p-m)<.01){if(Math.abs(h+f)<.1&&Math.abs(u+y)<.1&&Math.abs(p+m)<.1&&Math.abs(l+d+g-3)<.1)return this.set(1,0,0,0),this;t=Math.PI;let S=(l+1)/2,x=(d+1)/2,A=(g+1)/2,w=(h+f)/4,I=(u+y)/4,v=(p+m)/4;return S>x&&S>A?S<.01?(n=0,s=.707106781,r=.707106781):(n=Math.sqrt(S),s=w/n,r=I/n):x>A?x<.01?(n=.707106781,s=0,r=.707106781):(s=Math.sqrt(x),n=w/s,r=v/s):A<.01?(n=.707106781,s=.707106781,r=0):(r=Math.sqrt(A),n=I/r,s=v/r),this.set(n,s,r,t),this}let E=Math.sqrt((m-p)*(m-p)+(u-y)*(u-y)+(f-h)*(f-h));return Math.abs(E)<.001&&(E=1),this.x=(m-p)/E,this.y=(u-y)/E,this.z=(f-h)/E,this.w=Math.acos((l+d+g-1)/2),this}setFromMatrixPosition(e){let t=e.elements;return this.x=t[12],this.y=t[13],this.z=t[14],this.w=t[15],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this.w=Math.min(this.w,e.w),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this.w=Math.max(this.w,e.w),this}clamp(e,t){return this.x=et(this.x,e.x,t.x),this.y=et(this.y,e.y,t.y),this.z=et(this.z,e.z,t.z),this.w=et(this.w,e.w,t.w),this}clampScalar(e,t){return this.x=et(this.x,e,t),this.y=et(this.y,e,t),this.z=et(this.z,e,t),this.w=et(this.w,e,t),this}clampLength(e,t){let n=this.length();return this.divideScalar(n||1).multiplyScalar(et(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this.w=Math.floor(this.w),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this.w=Math.ceil(this.w),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this.w=Math.round(this.w),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this.w=Math.trunc(this.w),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this.w=-this.w,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z+this.w*e.w}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)+Math.abs(this.w)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this.z+=(e.z-this.z)*t,this.w+=(e.w-this.w)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this.z=e.z+(t.z-e.z)*n,this.w=e.w+(t.w-e.w)*n,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z&&e.w===this.w}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this.z=e[t+2],this.w=e[t+3],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e[t+2]=this.z,e[t+3]=this.w,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this.z=e.getZ(t),this.w=e.getW(t),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this.w=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z,yield this.w}};oh.prototype.isVector4=!0;var ft=oh,_a=class extends Fn{constructor(e=1,t=1,n={}){super(),n=Object.assign({generateMipmaps:!1,internalFormat:null,minFilter:bt,depthBuffer:!0,stencilBuffer:!1,resolveDepthBuffer:!0,resolveStencilBuffer:!0,depthTexture:null,samples:0,count:1,depth:1,multiview:!1,useArrayDepthTexture:!1},n),this.isRenderTarget=!0,this.width=e,this.height=t,this.depth=n.depth,this.scissor=new ft(0,0,e,t),this.scissorTest=!1,this.viewport=new ft(0,0,e,t),this.textures=[];let s={width:e,height:t,depth:n.depth},r=new Tt(s),o=n.count;for(let a=0;a<o;a++)this.textures[a]=r.clone(),this.textures[a].isRenderTargetTexture=!0,this.textures[a].renderTarget=this;this._setTextureOptions(n),this.depthBuffer=n.depthBuffer,this.stencilBuffer=n.stencilBuffer,this.resolveDepthBuffer=n.resolveDepthBuffer,this.resolveStencilBuffer=n.resolveStencilBuffer,this._depthTexture=null,this.depthTexture=n.depthTexture,this.samples=n.samples,this.multiview=n.multiview,this.useArrayDepthTexture=n.useArrayDepthTexture}_setTextureOptions(e={}){let t={minFilter:bt,generateMipmaps:!1,flipY:!1,internalFormat:null};e.mapping!==void 0&&(t.mapping=e.mapping),e.wrapS!==void 0&&(t.wrapS=e.wrapS),e.wrapT!==void 0&&(t.wrapT=e.wrapT),e.wrapR!==void 0&&(t.wrapR=e.wrapR),e.magFilter!==void 0&&(t.magFilter=e.magFilter),e.minFilter!==void 0&&(t.minFilter=e.minFilter),e.format!==void 0&&(t.format=e.format),e.type!==void 0&&(t.type=e.type),e.anisotropy!==void 0&&(t.anisotropy=e.anisotropy),e.colorSpace!==void 0&&(t.colorSpace=e.colorSpace),e.flipY!==void 0&&(t.flipY=e.flipY),e.generateMipmaps!==void 0&&(t.generateMipmaps=e.generateMipmaps),e.internalFormat!==void 0&&(t.internalFormat=e.internalFormat);for(let n=0;n<this.textures.length;n++)this.textures[n].setValues(t)}get texture(){return this.textures[0]}set texture(e){this.textures[0]=e}set depthTexture(e){this._depthTexture!==null&&(this._depthTexture.renderTarget=null),e!==null&&(e.renderTarget=this),this._depthTexture=e}get depthTexture(){return this._depthTexture}setSize(e,t,n=1){if(this.width!==e||this.height!==t||this.depth!==n){this.width=e,this.height=t,this.depth=n;for(let s=0,r=this.textures.length;s<r;s++)this.textures[s].image.width=e,this.textures[s].image.height=t,this.textures[s].image.depth=n,this.textures[s].isData3DTexture!==!0&&(this.textures[s].isArrayTexture=this.textures[s].image.depth>1);this.dispose()}this.viewport.set(0,0,e,t),this.scissor.set(0,0,e,t)}clone(){return new this.constructor().copy(this)}copy(e){this.width=e.width,this.height=e.height,this.depth=e.depth,this.scissor.copy(e.scissor),this.scissorTest=e.scissorTest,this.viewport.copy(e.viewport),this.textures.length=0;for(let t=0,n=e.textures.length;t<n;t++){this.textures[t]=e.textures[t].clone(),this.textures[t].isRenderTargetTexture=!0,this.textures[t].renderTarget=this;let s=Object.assign({},e.textures[t].image);this.textures[t].source=new Js(s)}return this.depthBuffer=e.depthBuffer,this.stencilBuffer=e.stencilBuffer,this.resolveDepthBuffer=e.resolveDepthBuffer,this.resolveStencilBuffer=e.resolveStencilBuffer,e.depthTexture!==null&&(this.depthTexture=e.depthTexture.clone()),this.samples=e.samples,this.multiview=e.multiview,this.useArrayDepthTexture=e.useArrayDepthTexture,this}dispose(){this.dispatchEvent({type:"dispose"})}},jt=class extends _a{constructor(e=1,t=1,n={}){super(e,t,n),this.isWebGLRenderTarget=!0}},Vr=class extends Tt{constructor(e=null,t=1,n=1,s=1){super(null),this.isDataArrayTexture=!0,this.image={data:e,width:t,height:n,depth:s},this.magFilter=St,this.minFilter=St,this.wrapR=Gt,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1,this.layerUpdates=new Set}addLayerUpdate(e){this.layerUpdates.add(e)}clearLayerUpdates(){this.layerUpdates.clear()}};var xa=class extends Tt{constructor(e=null,t=1,n=1,s=1){super(null),this.isData3DTexture=!0,this.image={data:e,width:t,height:n,depth:s},this.magFilter=St,this.minFilter=St,this.wrapR=Gt,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}};var za=class za{constructor(e,t,n,s,r,o,a,c,l,h,u,f,d,p,y,m){this.elements=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1],e!==void 0&&this.set(e,t,n,s,r,o,a,c,l,h,u,f,d,p,y,m)}set(e,t,n,s,r,o,a,c,l,h,u,f,d,p,y,m){let g=this.elements;return g[0]=e,g[4]=t,g[8]=n,g[12]=s,g[1]=r,g[5]=o,g[9]=a,g[13]=c,g[2]=l,g[6]=h,g[10]=u,g[14]=f,g[3]=d,g[7]=p,g[11]=y,g[15]=m,this}identity(){return this.set(1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1),this}clone(){return new za().fromArray(this.elements)}copy(e){let t=this.elements,n=e.elements;return t[0]=n[0],t[1]=n[1],t[2]=n[2],t[3]=n[3],t[4]=n[4],t[5]=n[5],t[6]=n[6],t[7]=n[7],t[8]=n[8],t[9]=n[9],t[10]=n[10],t[11]=n[11],t[12]=n[12],t[13]=n[13],t[14]=n[14],t[15]=n[15],this}copyPosition(e){let t=this.elements,n=e.elements;return t[12]=n[12],t[13]=n[13],t[14]=n[14],this}setFromMatrix3(e){let t=e.elements;return this.set(t[0],t[3],t[6],0,t[1],t[4],t[7],0,t[2],t[5],t[8],0,0,0,0,1),this}extractBasis(e,t,n){return this.determinantAffine()===0?(e.set(1,0,0),t.set(0,1,0),n.set(0,0,1),this):(e.setFromMatrixColumn(this,0),t.setFromMatrixColumn(this,1),n.setFromMatrixColumn(this,2),this)}makeBasis(e,t,n){return this.set(e.x,t.x,n.x,0,e.y,t.y,n.y,0,e.z,t.z,n.z,0,0,0,0,1),this}extractRotation(e){if(e.determinantAffine()===0)return this.identity();let t=this.elements,n=e.elements,s=1/Ns.setFromMatrixColumn(e,0).length(),r=1/Ns.setFromMatrixColumn(e,1).length(),o=1/Ns.setFromMatrixColumn(e,2).length();return t[0]=n[0]*s,t[1]=n[1]*s,t[2]=n[2]*s,t[3]=0,t[4]=n[4]*r,t[5]=n[5]*r,t[6]=n[6]*r,t[7]=0,t[8]=n[8]*o,t[9]=n[9]*o,t[10]=n[10]*o,t[11]=0,t[12]=0,t[13]=0,t[14]=0,t[15]=1,this}makeRotationFromEuler(e){let t=this.elements,n=e.x,s=e.y,r=e.z,o=Math.cos(n),a=Math.sin(n),c=Math.cos(s),l=Math.sin(s),h=Math.cos(r),u=Math.sin(r);if(e.order==="XYZ"){let f=o*h,d=o*u,p=a*h,y=a*u;t[0]=c*h,t[4]=-c*u,t[8]=l,t[1]=d+p*l,t[5]=f-y*l,t[9]=-a*c,t[2]=y-f*l,t[6]=p+d*l,t[10]=o*c}else if(e.order==="YXZ"){let f=c*h,d=c*u,p=l*h,y=l*u;t[0]=f+y*a,t[4]=p*a-d,t[8]=o*l,t[1]=o*u,t[5]=o*h,t[9]=-a,t[2]=d*a-p,t[6]=y+f*a,t[10]=o*c}else if(e.order==="ZXY"){let f=c*h,d=c*u,p=l*h,y=l*u;t[0]=f-y*a,t[4]=-o*u,t[8]=p+d*a,t[1]=d+p*a,t[5]=o*h,t[9]=y-f*a,t[2]=-o*l,t[6]=a,t[10]=o*c}else if(e.order==="ZYX"){let f=o*h,d=o*u,p=a*h,y=a*u;t[0]=c*h,t[4]=p*l-d,t[8]=f*l+y,t[1]=c*u,t[5]=y*l+f,t[9]=d*l-p,t[2]=-l,t[6]=a*c,t[10]=o*c}else if(e.order==="YZX"){let f=o*c,d=o*l,p=a*c,y=a*l;t[0]=c*h,t[4]=y-f*u,t[8]=p*u+d,t[1]=u,t[5]=o*h,t[9]=-a*h,t[2]=-l*h,t[6]=d*u+p,t[10]=f-y*u}else if(e.order==="XZY"){let f=o*c,d=o*l,p=a*c,y=a*l;t[0]=c*h,t[4]=-u,t[8]=l*h,t[1]=f*u+y,t[5]=o*h,t[9]=d*u-p,t[2]=p*u-d,t[6]=a*h,t[10]=y*u+f}return t[3]=0,t[7]=0,t[11]=0,t[12]=0,t[13]=0,t[14]=0,t[15]=1,this}makeRotationFromQuaternion(e){return this.compose(Mp,e,Sp)}lookAt(e,t,n){let s=this.elements;return pn.subVectors(e,t),pn.lengthSq()===0&&(pn.z=1),pn.normalize(),Li.crossVectors(n,pn),Li.lengthSq()===0&&(Math.abs(n.z)===1?pn.x+=1e-4:pn.z+=1e-4,pn.normalize(),Li.crossVectors(n,pn)),Li.normalize(),Io.crossVectors(pn,Li),s[0]=Li.x,s[4]=Io.x,s[8]=pn.x,s[1]=Li.y,s[5]=Io.y,s[9]=pn.y,s[2]=Li.z,s[6]=Io.z,s[10]=pn.z,this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,t){let n=e.elements,s=t.elements,r=this.elements,o=n[0],a=n[4],c=n[8],l=n[12],h=n[1],u=n[5],f=n[9],d=n[13],p=n[2],y=n[6],m=n[10],g=n[14],E=n[3],S=n[7],x=n[11],A=n[15],w=s[0],I=s[4],v=s[8],R=s[12],F=s[1],N=s[5],H=s[9],oe=s[13],ie=s[2],X=s[6],ee=s[10],Y=s[14],G=s[3],ge=s[7],ye=s[11],Me=s[15];return r[0]=o*w+a*F+c*ie+l*G,r[4]=o*I+a*N+c*X+l*ge,r[8]=o*v+a*H+c*ee+l*ye,r[12]=o*R+a*oe+c*Y+l*Me,r[1]=h*w+u*F+f*ie+d*G,r[5]=h*I+u*N+f*X+d*ge,r[9]=h*v+u*H+f*ee+d*ye,r[13]=h*R+u*oe+f*Y+d*Me,r[2]=p*w+y*F+m*ie+g*G,r[6]=p*I+y*N+m*X+g*ge,r[10]=p*v+y*H+m*ee+g*ye,r[14]=p*R+y*oe+m*Y+g*Me,r[3]=E*w+S*F+x*ie+A*G,r[7]=E*I+S*N+x*X+A*ge,r[11]=E*v+S*H+x*ee+A*ye,r[15]=E*R+S*oe+x*Y+A*Me,this}multiplyScalar(e){let t=this.elements;return t[0]*=e,t[4]*=e,t[8]*=e,t[12]*=e,t[1]*=e,t[5]*=e,t[9]*=e,t[13]*=e,t[2]*=e,t[6]*=e,t[10]*=e,t[14]*=e,t[3]*=e,t[7]*=e,t[11]*=e,t[15]*=e,this}determinant(){let e=this.elements,t=e[0],n=e[4],s=e[8],r=e[12],o=e[1],a=e[5],c=e[9],l=e[13],h=e[2],u=e[6],f=e[10],d=e[14],p=e[3],y=e[7],m=e[11],g=e[15],E=c*d-l*f,S=a*d-l*u,x=a*f-c*u,A=o*d-l*h,w=o*f-c*h,I=o*u-a*h;return t*(y*E-m*S+g*x)-n*(p*E-m*A+g*w)+s*(p*S-y*A+g*I)-r*(p*x-y*w+m*I)}determinantAffine(){let e=this.elements,t=e[0],n=e[4],s=e[8],r=e[1],o=e[5],a=e[9],c=e[2],l=e[6],h=e[10];return t*(o*h-a*l)-n*(r*h-a*c)+s*(r*l-o*c)}transpose(){let e=this.elements,t;return t=e[1],e[1]=e[4],e[4]=t,t=e[2],e[2]=e[8],e[8]=t,t=e[6],e[6]=e[9],e[9]=t,t=e[3],e[3]=e[12],e[12]=t,t=e[7],e[7]=e[13],e[13]=t,t=e[11],e[11]=e[14],e[14]=t,this}setPosition(e,t,n){let s=this.elements;return e.isVector3?(s[12]=e.x,s[13]=e.y,s[14]=e.z):(s[12]=e,s[13]=t,s[14]=n),this}invert(){let e=this.elements,t=e[0],n=e[1],s=e[2],r=e[3],o=e[4],a=e[5],c=e[6],l=e[7],h=e[8],u=e[9],f=e[10],d=e[11],p=e[12],y=e[13],m=e[14],g=e[15],E=t*a-n*o,S=t*c-s*o,x=t*l-r*o,A=n*c-s*a,w=n*l-r*a,I=s*l-r*c,v=h*y-u*p,R=h*m-f*p,F=h*g-d*p,N=u*m-f*y,H=u*g-d*y,oe=f*g-d*m,ie=E*oe-S*H+x*N+A*F-w*R+I*v;if(ie===0)return this.set(0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0);let X=1/ie;return e[0]=(a*oe-c*H+l*N)*X,e[1]=(s*H-n*oe-r*N)*X,e[2]=(y*I-m*w+g*A)*X,e[3]=(f*w-u*I-d*A)*X,e[4]=(c*F-o*oe-l*R)*X,e[5]=(t*oe-s*F+r*R)*X,e[6]=(m*x-p*I-g*S)*X,e[7]=(h*I-f*x+d*S)*X,e[8]=(o*H-a*F+l*v)*X,e[9]=(n*F-t*H-r*v)*X,e[10]=(p*w-y*x+g*E)*X,e[11]=(u*x-h*w-d*E)*X,e[12]=(a*R-o*N-c*v)*X,e[13]=(t*N-n*R+s*v)*X,e[14]=(y*S-p*A-m*E)*X,e[15]=(h*A-u*S+f*E)*X,this}scale(e){let t=this.elements,n=e.x,s=e.y,r=e.z;return t[0]*=n,t[4]*=s,t[8]*=r,t[1]*=n,t[5]*=s,t[9]*=r,t[2]*=n,t[6]*=s,t[10]*=r,t[3]*=n,t[7]*=s,t[11]*=r,this}getMaxScaleOnAxis(){let e=this.elements,t=e[0]*e[0]+e[1]*e[1]+e[2]*e[2],n=e[4]*e[4]+e[5]*e[5]+e[6]*e[6],s=e[8]*e[8]+e[9]*e[9]+e[10]*e[10];return Math.sqrt(Math.max(t,n,s))}makeTranslation(e,t,n){return e.isVector3?this.set(1,0,0,e.x,0,1,0,e.y,0,0,1,e.z,0,0,0,1):this.set(1,0,0,e,0,1,0,t,0,0,1,n,0,0,0,1),this}makeRotationX(e){let t=Math.cos(e),n=Math.sin(e);return this.set(1,0,0,0,0,t,-n,0,0,n,t,0,0,0,0,1),this}makeRotationY(e){let t=Math.cos(e),n=Math.sin(e);return this.set(t,0,n,0,0,1,0,0,-n,0,t,0,0,0,0,1),this}makeRotationZ(e){let t=Math.cos(e),n=Math.sin(e);return this.set(t,-n,0,0,n,t,0,0,0,0,1,0,0,0,0,1),this}makeRotationAxis(e,t){let n=Math.cos(t),s=Math.sin(t),r=1-n,o=e.x,a=e.y,c=e.z,l=r*o,h=r*a;return this.set(l*o+n,l*a-s*c,l*c+s*a,0,l*a+s*c,h*a+n,h*c-s*o,0,l*c-s*a,h*c+s*o,r*c*c+n,0,0,0,0,1),this}makeScale(e,t,n){return this.set(e,0,0,0,0,t,0,0,0,0,n,0,0,0,0,1),this}makeShear(e,t,n,s,r,o){return this.set(1,n,r,0,e,1,o,0,t,s,1,0,0,0,0,1),this}compose(e,t,n){let s=this.elements,r=t._x,o=t._y,a=t._z,c=t._w,l=r+r,h=o+o,u=a+a,f=r*l,d=r*h,p=r*u,y=o*h,m=o*u,g=a*u,E=c*l,S=c*h,x=c*u,A=n.x,w=n.y,I=n.z;return s[0]=(1-(y+g))*A,s[1]=(d+x)*A,s[2]=(p-S)*A,s[3]=0,s[4]=(d-x)*w,s[5]=(1-(f+g))*w,s[6]=(m+E)*w,s[7]=0,s[8]=(p+S)*I,s[9]=(m-E)*I,s[10]=(1-(f+y))*I,s[11]=0,s[12]=e.x,s[13]=e.y,s[14]=e.z,s[15]=1,this}decompose(e,t,n){let s=this.elements;e.x=s[12],e.y=s[13],e.z=s[14];let r=this.determinantAffine();if(r===0)return n.set(1,1,1),t.identity(),this;let o=Ns.set(s[0],s[1],s[2]).length(),a=Ns.set(s[4],s[5],s[6]).length(),c=Ns.set(s[8],s[9],s[10]).length();r<0&&(o=-o),In.copy(this);let l=1/o,h=1/a,u=1/c;return In.elements[0]*=l,In.elements[1]*=l,In.elements[2]*=l,In.elements[4]*=h,In.elements[5]*=h,In.elements[6]*=h,In.elements[8]*=u,In.elements[9]*=u,In.elements[10]*=u,t.setFromRotationMatrix(In),n.x=o,n.y=a,n.z=c,this}makePerspective(e,t,n,s,r,o,a=Un,c=!1){let l=this.elements,h=2*r/(t-e),u=2*r/(n-s),f=(t+e)/(t-e),d=(n+s)/(n-s),p,y;if(c)p=r/(o-r),y=o*r/(o-r);else if(a===Un)p=-(o+r)/(o-r),y=-2*o*r/(o-r);else if(a===Zs)p=-o/(o-r),y=-o*r/(o-r);else throw new Error("THREE.Matrix4.makePerspective(): Invalid coordinate system: "+a);return l[0]=h,l[4]=0,l[8]=f,l[12]=0,l[1]=0,l[5]=u,l[9]=d,l[13]=0,l[2]=0,l[6]=0,l[10]=p,l[14]=y,l[3]=0,l[7]=0,l[11]=-1,l[15]=0,this}makeOrthographic(e,t,n,s,r,o,a=Un,c=!1){let l=this.elements,h=2/(t-e),u=2/(n-s),f=-(t+e)/(t-e),d=-(n+s)/(n-s),p,y;if(c)p=1/(o-r),y=o/(o-r);else if(a===Un)p=-2/(o-r),y=-(o+r)/(o-r);else if(a===Zs)p=-1/(o-r),y=-r/(o-r);else throw new Error("THREE.Matrix4.makeOrthographic(): Invalid coordinate system: "+a);return l[0]=h,l[4]=0,l[8]=0,l[12]=f,l[1]=0,l[5]=u,l[9]=0,l[13]=d,l[2]=0,l[6]=0,l[10]=p,l[14]=y,l[3]=0,l[7]=0,l[11]=0,l[15]=1,this}equals(e){let t=this.elements,n=e.elements;for(let s=0;s<16;s++)if(t[s]!==n[s])return!1;return!0}fromArray(e,t=0){for(let n=0;n<16;n++)this.elements[n]=e[n+t];return this}toArray(e=[],t=0){let n=this.elements;return e[t]=n[0],e[t+1]=n[1],e[t+2]=n[2],e[t+3]=n[3],e[t+4]=n[4],e[t+5]=n[5],e[t+6]=n[6],e[t+7]=n[7],e[t+8]=n[8],e[t+9]=n[9],e[t+10]=n[10],e[t+11]=n[11],e[t+12]=n[12],e[t+13]=n[13],e[t+14]=n[14],e[t+15]=n[15],e}};za.prototype.isMatrix4=!0;var Je=za,Ns=new B,In=new Je,Mp=new B(0,0,0),Sp=new B(1,1,1),Li=new B,Io=new B,pn=new B,du=new Je,pu=new Kt,gi=class i{constructor(e=0,t=0,n=0,s=i.DEFAULT_ORDER){this.isEuler=!0,this._x=e,this._y=t,this._z=n,this._order=s}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get order(){return this._order}set order(e){this._order=e,this._onChangeCallback()}set(e,t,n,s=this._order){return this._x=e,this._y=t,this._z=n,this._order=s,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._order)}copy(e){return this._x=e._x,this._y=e._y,this._z=e._z,this._order=e._order,this._onChangeCallback(),this}setFromRotationMatrix(e,t=this._order,n=!0){let s=e.elements,r=s[0],o=s[4],a=s[8],c=s[1],l=s[5],h=s[9],u=s[2],f=s[6],d=s[10];switch(t){case"XYZ":this._y=Math.asin(et(a,-1,1)),Math.abs(a)<.9999999?(this._x=Math.atan2(-h,d),this._z=Math.atan2(-o,r)):(this._x=Math.atan2(f,l),this._z=0);break;case"YXZ":this._x=Math.asin(-et(h,-1,1)),Math.abs(h)<.9999999?(this._y=Math.atan2(a,d),this._z=Math.atan2(c,l)):(this._y=Math.atan2(-u,r),this._z=0);break;case"ZXY":this._x=Math.asin(et(f,-1,1)),Math.abs(f)<.9999999?(this._y=Math.atan2(-u,d),this._z=Math.atan2(-o,l)):(this._y=0,this._z=Math.atan2(c,r));break;case"ZYX":this._y=Math.asin(-et(u,-1,1)),Math.abs(u)<.9999999?(this._x=Math.atan2(f,d),this._z=Math.atan2(c,r)):(this._x=0,this._z=Math.atan2(-o,l));break;case"YZX":this._z=Math.asin(et(c,-1,1)),Math.abs(c)<.9999999?(this._x=Math.atan2(-h,l),this._y=Math.atan2(-u,r)):(this._x=0,this._y=Math.atan2(a,d));break;case"XZY":this._z=Math.asin(-et(o,-1,1)),Math.abs(o)<.9999999?(this._x=Math.atan2(f,l),this._y=Math.atan2(a,r)):(this._x=Math.atan2(-h,d),this._y=0);break;default:ke("Euler: .setFromRotationMatrix() encountered an unknown order: "+t)}return this._order=t,n===!0&&this._onChangeCallback(),this}setFromQuaternion(e,t,n){return du.makeRotationFromQuaternion(e),this.setFromRotationMatrix(du,t,n)}setFromVector3(e,t=this._order){return this.set(e.x,e.y,e.z,t)}reorder(e){return pu.setFromEuler(this),this.setFromQuaternion(pu,e)}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._order===this._order}fromArray(e){return this._x=e[0],this._y=e[1],this._z=e[2],e[3]!==void 0&&(this._order=e[3]),this._onChangeCallback(),this}toArray(e=[],t=0){return e[t]=this._x,e[t+1]=this._y,e[t+2]=this._z,e[t+3]=this._order,e}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._order}};gi.DEFAULT_ORDER="XYZ";var Gr=class{constructor(){this.mask=1}set(e){this.mask=(1<<e|0)>>>0}enable(e){this.mask|=1<<e|0}enableAll(){this.mask=-1}toggle(e){this.mask^=1<<e|0}disable(e){this.mask&=~(1<<e|0)}disableAll(){this.mask=0}test(e){return(this.mask&e.mask)!==0}isEnabled(e){return(this.mask&(1<<e|0))!==0}},Tp=0,mu=new B,Us=new Kt,li=new Je,Lo=new B,wr=new B,Ep=new B,Ap=new Kt,gu=new B(1,0,0),_u=new B(0,1,0),xu=new B(0,0,1),yu={type:"added"},wp={type:"removed"},Os={type:"childadded",child:null},Wc={type:"childremoved",child:null},At=class i extends Fn{constructor(){super(),this.isObject3D=!0,Object.defineProperty(this,"id",{value:Tp++}),this.uuid=Tn(),this.name="",this.type="Object3D",this.parent=null,this.children=[],this.up=i.DEFAULT_UP.clone();let e=new B,t=new gi,n=new Kt,s=new B(1,1,1);function r(){n.setFromEuler(t,!1)}function o(){t.setFromQuaternion(n,void 0,!1)}t._onChange(r),n._onChange(o),Object.defineProperties(this,{position:{configurable:!0,enumerable:!0,value:e},rotation:{configurable:!0,enumerable:!0,value:t},quaternion:{configurable:!0,enumerable:!0,value:n},scale:{configurable:!0,enumerable:!0,value:s},modelViewMatrix:{value:new Je},normalMatrix:{value:new Ze}}),this.matrix=new Je,this.matrixWorld=new Je,this.matrixAutoUpdate=i.DEFAULT_MATRIX_AUTO_UPDATE,this.matrixWorldAutoUpdate=i.DEFAULT_MATRIX_WORLD_AUTO_UPDATE,this.matrixWorldNeedsUpdate=!1,this.layers=new Gr,this.visible=!0,this.castShadow=!1,this.receiveShadow=!1,this.frustumCulled=!0,this.renderOrder=0,this.animations=[],this.customDepthMaterial=void 0,this.customDistanceMaterial=void 0,this.static=!1,this.userData={},this.pivot=null}onBeforeShadow(){}onAfterShadow(){}onBeforeRender(){}onAfterRender(){}applyMatrix4(e){this.matrixAutoUpdate&&this.updateMatrix(),this.matrix.premultiply(e),this.matrix.decompose(this.position,this.quaternion,this.scale)}applyQuaternion(e){return this.quaternion.premultiply(e),this}setRotationFromAxisAngle(e,t){this.quaternion.setFromAxisAngle(e,t)}setRotationFromEuler(e){this.quaternion.setFromEuler(e,!0)}setRotationFromMatrix(e){this.quaternion.setFromRotationMatrix(e)}setRotationFromQuaternion(e){this.quaternion.copy(e)}rotateOnAxis(e,t){return Us.setFromAxisAngle(e,t),this.quaternion.multiply(Us),this}rotateOnWorldAxis(e,t){return Us.setFromAxisAngle(e,t),this.quaternion.premultiply(Us),this}rotateX(e){return this.rotateOnAxis(gu,e)}rotateY(e){return this.rotateOnAxis(_u,e)}rotateZ(e){return this.rotateOnAxis(xu,e)}translateOnAxis(e,t){return mu.copy(e).applyQuaternion(this.quaternion),this.position.add(mu.multiplyScalar(t)),this}translateX(e){return this.translateOnAxis(gu,e)}translateY(e){return this.translateOnAxis(_u,e)}translateZ(e){return this.translateOnAxis(xu,e)}localToWorld(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(this.matrixWorld)}worldToLocal(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(li.copy(this.matrixWorld).invert())}lookAt(e,t,n){e.isVector3?Lo.copy(e):Lo.set(e,t,n);let s=this.parent;this.updateWorldMatrix(!0,!1),wr.setFromMatrixPosition(this.matrixWorld),this.isCamera||this.isLight?li.lookAt(wr,Lo,this.up):li.lookAt(Lo,wr,this.up),this.quaternion.setFromRotationMatrix(li),s&&(li.extractRotation(s.matrixWorld),Us.setFromRotationMatrix(li),this.quaternion.premultiply(Us.invert()))}add(e){if(arguments.length>1){for(let t=0;t<arguments.length;t++)this.add(arguments[t]);return this}return e===this?(Ke("Object3D.add: object can't be added as a child of itself.",e),this):(e&&e.isObject3D?(e.removeFromParent(),e.parent=this,this.children.push(e),e.dispatchEvent(yu),Os.child=e,this.dispatchEvent(Os),Os.child=null):Ke("Object3D.add: object not an instance of THREE.Object3D.",e),this)}remove(e){if(arguments.length>1){for(let n=0;n<arguments.length;n++)this.remove(arguments[n]);return this}let t=this.children.indexOf(e);return t!==-1&&(e.parent=null,this.children.splice(t,1),e.dispatchEvent(wp),Wc.child=e,this.dispatchEvent(Wc),Wc.child=null),this}removeFromParent(){let e=this.parent;return e!==null&&e.remove(this),this}clear(){return this.remove(...this.children)}attach(e){return this.updateWorldMatrix(!0,!1),li.copy(this.matrixWorld).invert(),e.parent!==null&&(e.parent.updateWorldMatrix(!0,!1),li.multiply(e.parent.matrixWorld)),e.applyMatrix4(li),e.removeFromParent(),e.parent=this,this.children.push(e),e.updateWorldMatrix(!1,!0),e.dispatchEvent(yu),Os.child=e,this.dispatchEvent(Os),Os.child=null,this}getObjectById(e){return this.getObjectByProperty("id",e)}getObjectByName(e){return this.getObjectByProperty("name",e)}getObjectByProperty(e,t){if(this[e]===t)return this;for(let n=0,s=this.children.length;n<s;n++){let o=this.children[n].getObjectByProperty(e,t);if(o!==void 0)return o}}getObjectsByProperty(e,t,n=[]){this[e]===t&&n.push(this);let s=this.children;for(let r=0,o=s.length;r<o;r++)s[r].getObjectsByProperty(e,t,n);return n}getWorldPosition(e){return this.updateWorldMatrix(!0,!1),e.setFromMatrixPosition(this.matrixWorld)}getWorldQuaternion(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(wr,e,Ep),e}getWorldScale(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(wr,Ap,e),e}getWorldDirection(e){this.updateWorldMatrix(!0,!1);let t=this.matrixWorld.elements;return e.set(t[8],t[9],t[10]).normalize()}raycast(){}traverse(e){e(this);let t=this.children;for(let n=0,s=t.length;n<s;n++)t[n].traverse(e)}traverseVisible(e){if(this.visible===!1)return;e(this);let t=this.children;for(let n=0,s=t.length;n<s;n++)t[n].traverseVisible(e)}traverseAncestors(e){let t=this.parent;t!==null&&(e(t),t.traverseAncestors(e))}updateMatrix(){this.matrix.compose(this.position,this.quaternion,this.scale);let e=this.pivot;if(e!==null){let t=e.x,n=e.y,s=e.z,r=this.matrix.elements;r[12]+=t-r[0]*t-r[4]*n-r[8]*s,r[13]+=n-r[1]*t-r[5]*n-r[9]*s,r[14]+=s-r[2]*t-r[6]*n-r[10]*s}this.matrixWorldNeedsUpdate=!0}updateMatrixWorld(e){this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||e)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,e=!0);let t=this.children;for(let n=0,s=t.length;n<s;n++)t[n].updateMatrixWorld(e)}updateWorldMatrix(e,t,n=!1){let s=this.parent;if(e===!0&&s!==null&&s.updateWorldMatrix(!0,!1),this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||n)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,n=!0),t===!0){let r=this.children;for(let o=0,a=r.length;o<a;o++)r[o].updateWorldMatrix(!1,!0,n)}}toJSON(e){let t=e===void 0||typeof e=="string",n={};t&&(e={geometries:{},materials:{},textures:{},images:{},shapes:{},skeletons:{},animations:{},nodes:{}},n.metadata={version:4.7,type:"Object",generator:"Object3D.toJSON"});let s={};s.uuid=this.uuid,s.type=this.type,this.name!==""&&(s.name=this.name),this.castShadow===!0&&(s.castShadow=!0),this.receiveShadow===!0&&(s.receiveShadow=!0),this.visible===!1&&(s.visible=!1),this.frustumCulled===!1&&(s.frustumCulled=!1),this.renderOrder!==0&&(s.renderOrder=this.renderOrder),this.static!==!1&&(s.static=this.static),Object.keys(this.userData).length>0&&(s.userData=this.userData),s.layers=this.layers.mask,s.matrix=this.matrix.toArray(),s.up=this.up.toArray(),this.pivot!==null&&(s.pivot=this.pivot.toArray()),this.matrixAutoUpdate===!1&&(s.matrixAutoUpdate=!1),this.morphTargetDictionary!==void 0&&(s.morphTargetDictionary=Object.assign({},this.morphTargetDictionary)),this.morphTargetInfluences!==void 0&&(s.morphTargetInfluences=this.morphTargetInfluences.slice()),this.isInstancedMesh&&(s.type="InstancedMesh",s.count=this.count,s.instanceMatrix=this.instanceMatrix.toJSON(),this.instanceColor!==null&&(s.instanceColor=this.instanceColor.toJSON())),this.isBatchedMesh&&(s.type="BatchedMesh",s.perObjectFrustumCulled=this.perObjectFrustumCulled,s.sortObjects=this.sortObjects,s.drawRanges=this._drawRanges,s.reservedRanges=this._reservedRanges,s.geometryInfo=this._geometryInfo.map(a=>({...a,boundingBox:a.boundingBox?a.boundingBox.toJSON():void 0,boundingSphere:a.boundingSphere?a.boundingSphere.toJSON():void 0})),s.instanceInfo=this._instanceInfo.map(a=>({...a})),s.availableInstanceIds=this._availableInstanceIds.slice(),s.availableGeometryIds=this._availableGeometryIds.slice(),s.nextIndexStart=this._nextIndexStart,s.nextVertexStart=this._nextVertexStart,s.geometryCount=this._geometryCount,s.maxInstanceCount=this._maxInstanceCount,s.maxVertexCount=this._maxVertexCount,s.maxIndexCount=this._maxIndexCount,s.geometryInitialized=this._geometryInitialized,s.matricesTexture=this._matricesTexture.toJSON(e),s.indirectTexture=this._indirectTexture.toJSON(e),this._colorsTexture!==null&&(s.colorsTexture=this._colorsTexture.toJSON(e)),this.boundingSphere!==null&&(s.boundingSphere=this.boundingSphere.toJSON()),this.boundingBox!==null&&(s.boundingBox=this.boundingBox.toJSON()));function r(a,c){return a[c.uuid]===void 0&&(a[c.uuid]=c.toJSON(e)),c.uuid}if(this.isScene)this.background&&(this.background.isColor?s.background=this.background.toJSON():this.background.isTexture&&(s.background=this.background.toJSON(e).uuid)),this.environment&&this.environment.isTexture&&this.environment.isRenderTargetTexture!==!0&&(s.environment=this.environment.toJSON(e).uuid);else if(this.isMesh||this.isLine||this.isPoints){s.geometry=r(e.geometries,this.geometry);let a=this.geometry.parameters;if(a!==void 0&&a.shapes!==void 0){let c=a.shapes;if(Array.isArray(c))for(let l=0,h=c.length;l<h;l++){let u=c[l];r(e.shapes,u)}else r(e.shapes,c)}}if(this.isSkinnedMesh&&(s.bindMode=this.bindMode,s.bindMatrix=this.bindMatrix.toArray(),this.skeleton!==void 0&&(r(e.skeletons,this.skeleton),s.skeleton=this.skeleton.uuid)),this.material!==void 0)if(Array.isArray(this.material)){let a=[];for(let c=0,l=this.material.length;c<l;c++)a.push(r(e.materials,this.material[c]));s.material=a}else s.material=r(e.materials,this.material);if(this.children.length>0){s.children=[];for(let a=0;a<this.children.length;a++)s.children.push(this.children[a].toJSON(e).object)}if(this.animations.length>0){s.animations=[];for(let a=0;a<this.animations.length;a++){let c=this.animations[a];s.animations.push(r(e.animations,c))}}if(t){let a=o(e.geometries),c=o(e.materials),l=o(e.textures),h=o(e.images),u=o(e.shapes),f=o(e.skeletons),d=o(e.animations),p=o(e.nodes);a.length>0&&(n.geometries=a),c.length>0&&(n.materials=c),l.length>0&&(n.textures=l),h.length>0&&(n.images=h),u.length>0&&(n.shapes=u),f.length>0&&(n.skeletons=f),d.length>0&&(n.animations=d),p.length>0&&(n.nodes=p)}return n.object=s,n;function o(a){let c=[];for(let l in a){let h=a[l];delete h.metadata,c.push(h)}return c}}clone(e){return new this.constructor().copy(this,e)}copy(e,t=!0){if(this.name=e.name,this.up.copy(e.up),this.position.copy(e.position),this.rotation.order=e.rotation.order,this.quaternion.copy(e.quaternion),this.scale.copy(e.scale),this.pivot=e.pivot!==null?e.pivot.clone():null,this.matrix.copy(e.matrix),this.matrixWorld.copy(e.matrixWorld),this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrixWorldAutoUpdate=e.matrixWorldAutoUpdate,this.matrixWorldNeedsUpdate=e.matrixWorldNeedsUpdate,this.layers.mask=e.layers.mask,this.visible=e.visible,this.castShadow=e.castShadow,this.receiveShadow=e.receiveShadow,this.frustumCulled=e.frustumCulled,this.renderOrder=e.renderOrder,this.static=e.static,this.animations=e.animations.slice(),this.userData=JSON.parse(JSON.stringify(e.userData)),t===!0)for(let n=0;n<e.children.length;n++){let s=e.children[n];this.add(s.clone())}return this}};At.DEFAULT_UP=new B(0,1,0);At.DEFAULT_MATRIX_AUTO_UPDATE=!0;At.DEFAULT_MATRIX_WORLD_AUTO_UPDATE=!0;var Ut=class extends At{constructor(){super(),this.isGroup=!0,this.type="Group"}},Rp={type:"move"},$s=class{constructor(){this._targetRay=null,this._grip=null,this._hand=null}getHandSpace(){return this._hand===null&&(this._hand=new Ut,this._hand.matrixAutoUpdate=!1,this._hand.visible=!1,this._hand.joints={},this._hand.inputState={pinching:!1}),this._hand}getTargetRaySpace(){return this._targetRay===null&&(this._targetRay=new Ut,this._targetRay.matrixAutoUpdate=!1,this._targetRay.visible=!1,this._targetRay.hasLinearVelocity=!1,this._targetRay.linearVelocity=new B,this._targetRay.hasAngularVelocity=!1,this._targetRay.angularVelocity=new B),this._targetRay}getGripSpace(){return this._grip===null&&(this._grip=new Ut,this._grip.matrixAutoUpdate=!1,this._grip.visible=!1,this._grip.hasLinearVelocity=!1,this._grip.linearVelocity=new B,this._grip.hasAngularVelocity=!1,this._grip.angularVelocity=new B,this._grip.eventsEnabled=!1),this._grip}dispatchEvent(e){return this._targetRay!==null&&this._targetRay.dispatchEvent(e),this._grip!==null&&this._grip.dispatchEvent(e),this._hand!==null&&this._hand.dispatchEvent(e),this}connect(e){if(e&&e.hand){let t=this._hand;if(t)for(let n of e.hand.values())this._getHandJoint(t,n)}return this.dispatchEvent({type:"connected",data:e}),this}disconnect(e){return this.dispatchEvent({type:"disconnected",data:e}),this._targetRay!==null&&(this._targetRay.visible=!1),this._grip!==null&&(this._grip.visible=!1),this._hand!==null&&(this._hand.visible=!1),this}update(e,t,n){let s=null,r=null,o=null,a=this._targetRay,c=this._grip,l=this._hand;if(e&&t.session.visibilityState!=="visible-blurred"){if(l&&e.hand){o=!0;for(let y of e.hand.values()){let m=t.getJointPose(y,n),g=this._getHandJoint(l,y);m!==null&&(g.matrix.fromArray(m.transform.matrix),g.matrix.decompose(g.position,g.rotation,g.scale),g.matrixWorldNeedsUpdate=!0,g.jointRadius=m.radius),g.visible=m!==null}let h=l.joints["index-finger-tip"],u=l.joints["thumb-tip"],f=h.position.distanceTo(u.position),d=.02,p=.005;l.inputState.pinching&&f>d+p?(l.inputState.pinching=!1,this.dispatchEvent({type:"pinchend",handedness:e.handedness,target:this})):!l.inputState.pinching&&f<=d-p&&(l.inputState.pinching=!0,this.dispatchEvent({type:"pinchstart",handedness:e.handedness,target:this}))}else c!==null&&e.gripSpace&&(r=t.getPose(e.gripSpace,n),r!==null&&(c.matrix.fromArray(r.transform.matrix),c.matrix.decompose(c.position,c.rotation,c.scale),c.matrixWorldNeedsUpdate=!0,r.linearVelocity?(c.hasLinearVelocity=!0,c.linearVelocity.copy(r.linearVelocity)):c.hasLinearVelocity=!1,r.angularVelocity?(c.hasAngularVelocity=!0,c.angularVelocity.copy(r.angularVelocity)):c.hasAngularVelocity=!1,c.eventsEnabled&&c.dispatchEvent({type:"gripUpdated",data:e,target:this})));a!==null&&(s=t.getPose(e.targetRaySpace,n),s===null&&r!==null&&(s=r),s!==null&&(a.matrix.fromArray(s.transform.matrix),a.matrix.decompose(a.position,a.rotation,a.scale),a.matrixWorldNeedsUpdate=!0,s.linearVelocity?(a.hasLinearVelocity=!0,a.linearVelocity.copy(s.linearVelocity)):a.hasLinearVelocity=!1,s.angularVelocity?(a.hasAngularVelocity=!0,a.angularVelocity.copy(s.angularVelocity)):a.hasAngularVelocity=!1,this.dispatchEvent(Rp)))}return a!==null&&(a.visible=s!==null),c!==null&&(c.visible=r!==null),l!==null&&(l.visible=o!==null),this}_getHandJoint(e,t){if(e.joints[t.jointName]===void 0){let n=new Ut;n.matrixAutoUpdate=!1,n.visible=!1,e.joints[t.jointName]=n,e.add(n)}return e.joints[t.jointName]}},Uf={aliceblue:15792383,antiquewhite:16444375,aqua:65535,aquamarine:8388564,azure:15794175,beige:16119260,bisque:16770244,black:0,blanchedalmond:16772045,blue:255,blueviolet:9055202,brown:10824234,burlywood:14596231,cadetblue:6266528,chartreuse:8388352,chocolate:13789470,coral:16744272,cornflowerblue:6591981,cornsilk:16775388,crimson:14423100,cyan:65535,darkblue:139,darkcyan:35723,darkgoldenrod:12092939,darkgray:11119017,darkgreen:25600,darkgrey:11119017,darkkhaki:12433259,darkmagenta:9109643,darkolivegreen:5597999,darkorange:16747520,darkorchid:10040012,darkred:9109504,darksalmon:15308410,darkseagreen:9419919,darkslateblue:4734347,darkslategray:3100495,darkslategrey:3100495,darkturquoise:52945,darkviolet:9699539,deeppink:16716947,deepskyblue:49151,dimgray:6908265,dimgrey:6908265,dodgerblue:2003199,firebrick:11674146,floralwhite:16775920,forestgreen:2263842,fuchsia:16711935,gainsboro:14474460,ghostwhite:16316671,gold:16766720,goldenrod:14329120,gray:8421504,green:32768,greenyellow:11403055,grey:8421504,honeydew:15794160,hotpink:16738740,indianred:13458524,indigo:4915330,ivory:16777200,khaki:15787660,lavender:15132410,lavenderblush:16773365,lawngreen:8190976,lemonchiffon:16775885,lightblue:11393254,lightcoral:15761536,lightcyan:14745599,lightgoldenrodyellow:16448210,lightgray:13882323,lightgreen:9498256,lightgrey:13882323,lightpink:16758465,lightsalmon:16752762,lightseagreen:2142890,lightskyblue:8900346,lightslategray:7833753,lightslategrey:7833753,lightsteelblue:11584734,lightyellow:16777184,lime:65280,limegreen:3329330,linen:16445670,magenta:16711935,maroon:8388608,mediumaquamarine:6737322,mediumblue:205,mediumorchid:12211667,mediumpurple:9662683,mediumseagreen:3978097,mediumslateblue:8087790,mediumspringgreen:64154,mediumturquoise:4772300,mediumvioletred:13047173,midnightblue:1644912,mintcream:16121850,mistyrose:16770273,moccasin:16770229,navajowhite:16768685,navy:128,oldlace:16643558,olive:8421376,olivedrab:7048739,orange:16753920,orangered:16729344,orchid:14315734,palegoldenrod:15657130,palegreen:10025880,paleturquoise:11529966,palevioletred:14381203,papayawhip:16773077,peachpuff:16767673,peru:13468991,pink:16761035,plum:14524637,powderblue:11591910,purple:8388736,rebeccapurple:6697881,red:16711680,rosybrown:12357519,royalblue:4286945,saddlebrown:9127187,salmon:16416882,sandybrown:16032864,seagreen:3050327,seashell:16774638,sienna:10506797,silver:12632256,skyblue:8900331,slateblue:6970061,slategray:7372944,slategrey:7372944,snow:16775930,springgreen:65407,steelblue:4620980,tan:13808780,teal:32896,thistle:14204888,tomato:16737095,turquoise:4251856,violet:15631086,wheat:16113331,white:16777215,whitesmoke:16119285,yellow:16776960,yellowgreen:10145074},Di={h:0,s:0,l:0},Do={h:0,s:0,l:0};function Xc(i,e,t){return t<0&&(t+=1),t>1&&(t-=1),t<1/6?i+(e-i)*6*t:t<1/2?e:t<2/3?i+(e-i)*6*(2/3-t):i}var Ge=class{constructor(e,t,n){return this.isColor=!0,this.r=1,this.g=1,this.b=1,this.set(e,t,n)}set(e,t,n){if(t===void 0&&n===void 0){let s=e;s&&s.isColor?this.copy(s):typeof s=="number"?this.setHex(s):typeof s=="string"&&this.setStyle(s)}else this.setRGB(e,t,n);return this}setScalar(e){return this.r=e,this.g=e,this.b=e,this}setHex(e,t=at){return e=Math.floor(e),this.r=(e>>16&255)/255,this.g=(e>>8&255)/255,this.b=(e&255)/255,Qe.colorSpaceToWorking(this,t),this}setRGB(e,t,n,s=Qe.workingColorSpace){return this.r=e,this.g=t,this.b=n,Qe.colorSpaceToWorking(this,s),this}setHSL(e,t,n,s=Qe.workingColorSpace){if(e=Jl(e,1),t=et(t,0,1),n=et(n,0,1),t===0)this.r=this.g=this.b=n;else{let r=n<=.5?n*(1+t):n+t-n*t,o=2*n-r;this.r=Xc(o,r,e+1/3),this.g=Xc(o,r,e),this.b=Xc(o,r,e-1/3)}return Qe.colorSpaceToWorking(this,s),this}setStyle(e,t=at){function n(r){r!==void 0&&parseFloat(r)<1&&ke("Color: Alpha component of "+e+" will be ignored.")}let s;if(s=/^(\w+)\(([^\)]*)\)/.exec(e)){let r,o=s[1],a=s[2];switch(o){case"rgb":case"rgba":if(r=/^\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(a))return n(r[4]),this.setRGB(Math.min(255,parseInt(r[1],10))/255,Math.min(255,parseInt(r[2],10))/255,Math.min(255,parseInt(r[3],10))/255,t);if(r=/^\s*(\d+)\%\s*,\s*(\d+)\%\s*,\s*(\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(a))return n(r[4]),this.setRGB(Math.min(100,parseInt(r[1],10))/100,Math.min(100,parseInt(r[2],10))/100,Math.min(100,parseInt(r[3],10))/100,t);break;case"hsl":case"hsla":if(r=/^\s*(\d*\.?\d+)\s*,\s*(\d*\.?\d+)\%\s*,\s*(\d*\.?\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(a))return n(r[4]),this.setHSL(parseFloat(r[1])/360,parseFloat(r[2])/100,parseFloat(r[3])/100,t);break;default:ke("Color: Unknown color model "+e)}}else if(s=/^\#([A-Fa-f\d]+)$/.exec(e)){let r=s[1],o=r.length;if(o===3)return this.setRGB(parseInt(r.charAt(0),16)/15,parseInt(r.charAt(1),16)/15,parseInt(r.charAt(2),16)/15,t);if(o===6)return this.setHex(parseInt(r,16),t);ke("Color: Invalid hex color "+e)}else if(e&&e.length>0)return this.setColorName(e,t);return this}setColorName(e,t=at){let n=Uf[e.toLowerCase()];return n!==void 0?this.setHex(n,t):ke("Color: Unknown color "+e),this}clone(){return new this.constructor(this.r,this.g,this.b)}copy(e){return this.r=e.r,this.g=e.g,this.b=e.b,this}copySRGBToLinear(e){return this.r=mi(e.r),this.g=mi(e.g),this.b=mi(e.b),this}copyLinearToSRGB(e){return this.r=Ys(e.r),this.g=Ys(e.g),this.b=Ys(e.b),this}convertSRGBToLinear(){return this.copySRGBToLinear(this),this}convertLinearToSRGB(){return this.copyLinearToSRGB(this),this}getHex(e=at){return Qe.workingToColorSpace(Zt.copy(this),e),Math.round(et(Zt.r*255,0,255))*65536+Math.round(et(Zt.g*255,0,255))*256+Math.round(et(Zt.b*255,0,255))}getHexString(e=at){return("000000"+this.getHex(e).toString(16)).slice(-6)}getHSL(e,t=Qe.workingColorSpace){Qe.workingToColorSpace(Zt.copy(this),t);let n=Zt.r,s=Zt.g,r=Zt.b,o=Math.max(n,s,r),a=Math.min(n,s,r),c,l,h=(a+o)/2;if(a===o)c=0,l=0;else{let u=o-a;switch(l=h<=.5?u/(o+a):u/(2-o-a),o){case n:c=(s-r)/u+(s<r?6:0);break;case s:c=(r-n)/u+2;break;case r:c=(n-s)/u+4;break}c/=6}return e.h=c,e.s=l,e.l=h,e}getRGB(e,t=Qe.workingColorSpace){return Qe.workingToColorSpace(Zt.copy(this),t),e.r=Zt.r,e.g=Zt.g,e.b=Zt.b,e}getStyle(e=at){Qe.workingToColorSpace(Zt.copy(this),e);let t=Zt.r,n=Zt.g,s=Zt.b;return e!==at?`color(${e} ${t.toFixed(3)} ${n.toFixed(3)} ${s.toFixed(3)})`:`rgb(${Math.round(t*255)},${Math.round(n*255)},${Math.round(s*255)})`}offsetHSL(e,t,n){return this.getHSL(Di),this.setHSL(Di.h+e,Di.s+t,Di.l+n)}add(e){return this.r+=e.r,this.g+=e.g,this.b+=e.b,this}addColors(e,t){return this.r=e.r+t.r,this.g=e.g+t.g,this.b=e.b+t.b,this}addScalar(e){return this.r+=e,this.g+=e,this.b+=e,this}sub(e){return this.r=Math.max(0,this.r-e.r),this.g=Math.max(0,this.g-e.g),this.b=Math.max(0,this.b-e.b),this}multiply(e){return this.r*=e.r,this.g*=e.g,this.b*=e.b,this}multiplyScalar(e){return this.r*=e,this.g*=e,this.b*=e,this}lerp(e,t){return this.r+=(e.r-this.r)*t,this.g+=(e.g-this.g)*t,this.b+=(e.b-this.b)*t,this}lerpColors(e,t,n){return this.r=e.r+(t.r-e.r)*n,this.g=e.g+(t.g-e.g)*n,this.b=e.b+(t.b-e.b)*n,this}lerpHSL(e,t){this.getHSL(Di),e.getHSL(Do);let n=Fr(Di.h,Do.h,t),s=Fr(Di.s,Do.s,t),r=Fr(Di.l,Do.l,t);return this.setHSL(n,s,r),this}setFromVector3(e){return this.r=e.x,this.g=e.y,this.b=e.z,this}applyMatrix3(e){let t=this.r,n=this.g,s=this.b,r=e.elements;return this.r=r[0]*t+r[3]*n+r[6]*s,this.g=r[1]*t+r[4]*n+r[7]*s,this.b=r[2]*t+r[5]*n+r[8]*s,this}equals(e){return e.r===this.r&&e.g===this.g&&e.b===this.b}fromArray(e,t=0){return this.r=e[t],this.g=e[t+1],this.b=e[t+2],this}toArray(e=[],t=0){return e[t]=this.r,e[t+1]=this.g,e[t+2]=this.b,e}fromBufferAttribute(e,t){return this.r=e.getX(t),this.g=e.getY(t),this.b=e.getZ(t),this}toJSON(){return this.getHex()}*[Symbol.iterator](){yield this.r,yield this.g,yield this.b}},Zt=new Ge;Ge.NAMES=Uf;var Bn=class extends At{constructor(){super(),this.isScene=!0,this.type="Scene",this.background=null,this.environment=null,this.fog=null,this.backgroundBlurriness=0,this.backgroundIntensity=1,this.backgroundRotation=new gi,this.environmentIntensity=1,this.environmentRotation=new gi,this.overrideMaterial=null,typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}copy(e,t){return super.copy(e,t),e.background!==null&&(this.background=e.background.clone()),e.environment!==null&&(this.environment=e.environment.clone()),e.fog!==null&&(this.fog=e.fog.clone()),this.backgroundBlurriness=e.backgroundBlurriness,this.backgroundIntensity=e.backgroundIntensity,this.backgroundRotation.copy(e.backgroundRotation),this.environmentIntensity=e.environmentIntensity,this.environmentRotation.copy(e.environmentRotation),e.overrideMaterial!==null&&(this.overrideMaterial=e.overrideMaterial.clone()),this.matrixAutoUpdate=e.matrixAutoUpdate,this}toJSON(e){let t=super.toJSON(e);return this.fog!==null&&(t.object.fog=this.fog.toJSON()),this.backgroundBlurriness>0&&(t.object.backgroundBlurriness=this.backgroundBlurriness),this.backgroundIntensity!==1&&(t.object.backgroundIntensity=this.backgroundIntensity),t.object.backgroundRotation=this.backgroundRotation.toArray(),this.environmentIntensity!==1&&(t.object.environmentIntensity=this.environmentIntensity),t.object.environmentRotation=this.environmentRotation.toArray(),t}},Ln=new B,hi=new B,qc=new B,ui=new B,Fs=new B,Bs=new B,vu=new B,Yc=new B,Zc=new B,Kc=new B,jc=new ft,Jc=new ft,$c=new ft,Bi=class i{constructor(e=new B,t=new B,n=new B){this.a=e,this.b=t,this.c=n}static getNormal(e,t,n,s){s.subVectors(n,t),Ln.subVectors(e,t),s.cross(Ln);let r=s.lengthSq();return r>0?s.multiplyScalar(1/Math.sqrt(r)):s.set(0,0,0)}static getBarycoord(e,t,n,s,r){Ln.subVectors(s,t),hi.subVectors(n,t),qc.subVectors(e,t);let o=Ln.dot(Ln),a=Ln.dot(hi),c=Ln.dot(qc),l=hi.dot(hi),h=hi.dot(qc),u=o*l-a*a;if(u===0)return r.set(0,0,0),null;let f=1/u,d=(l*c-a*h)*f,p=(o*h-a*c)*f;return r.set(1-d-p,p,d)}static containsPoint(e,t,n,s){return this.getBarycoord(e,t,n,s,ui)===null?!1:ui.x>=0&&ui.y>=0&&ui.x+ui.y<=1}static getInterpolation(e,t,n,s,r,o,a,c){return this.getBarycoord(e,t,n,s,ui)===null?(c.x=0,c.y=0,"z"in c&&(c.z=0),"w"in c&&(c.w=0),null):(c.setScalar(0),c.addScaledVector(r,ui.x),c.addScaledVector(o,ui.y),c.addScaledVector(a,ui.z),c)}static getInterpolatedAttribute(e,t,n,s,r,o){return jc.setScalar(0),Jc.setScalar(0),$c.setScalar(0),jc.fromBufferAttribute(e,t),Jc.fromBufferAttribute(e,n),$c.fromBufferAttribute(e,s),o.setScalar(0),o.addScaledVector(jc,r.x),o.addScaledVector(Jc,r.y),o.addScaledVector($c,r.z),o}static isFrontFacing(e,t,n,s){return Ln.subVectors(n,t),hi.subVectors(e,t),Ln.cross(hi).dot(s)<0}set(e,t,n){return this.a.copy(e),this.b.copy(t),this.c.copy(n),this}setFromPointsAndIndices(e,t,n,s){return this.a.copy(e[t]),this.b.copy(e[n]),this.c.copy(e[s]),this}setFromAttributeAndIndices(e,t,n,s){return this.a.fromBufferAttribute(e,t),this.b.fromBufferAttribute(e,n),this.c.fromBufferAttribute(e,s),this}clone(){return new this.constructor().copy(this)}copy(e){return this.a.copy(e.a),this.b.copy(e.b),this.c.copy(e.c),this}getArea(){return Ln.subVectors(this.c,this.b),hi.subVectors(this.a,this.b),Ln.cross(hi).length()*.5}getMidpoint(e){return e.addVectors(this.a,this.b).add(this.c).multiplyScalar(1/3)}getNormal(e){return i.getNormal(this.a,this.b,this.c,e)}getPlane(e){return e.setFromCoplanarPoints(this.a,this.b,this.c)}getBarycoord(e,t){return i.getBarycoord(e,this.a,this.b,this.c,t)}getInterpolation(e,t,n,s,r){return i.getInterpolation(e,this.a,this.b,this.c,t,n,s,r)}containsPoint(e){return i.containsPoint(e,this.a,this.b,this.c)}isFrontFacing(e){return i.isFrontFacing(this.a,this.b,this.c,e)}intersectsBox(e){return e.intersectsTriangle(this)}closestPointToPoint(e,t){let n=this.a,s=this.b,r=this.c,o,a;Fs.subVectors(s,n),Bs.subVectors(r,n),Yc.subVectors(e,n);let c=Fs.dot(Yc),l=Bs.dot(Yc);if(c<=0&&l<=0)return t.copy(n);Zc.subVectors(e,s);let h=Fs.dot(Zc),u=Bs.dot(Zc);if(h>=0&&u<=h)return t.copy(s);let f=c*u-h*l;if(f<=0&&c>=0&&h<=0)return o=c/(c-h),t.copy(n).addScaledVector(Fs,o);Kc.subVectors(e,r);let d=Fs.dot(Kc),p=Bs.dot(Kc);if(p>=0&&d<=p)return t.copy(r);let y=d*l-c*p;if(y<=0&&l>=0&&p<=0)return a=l/(l-p),t.copy(n).addScaledVector(Bs,a);let m=h*p-d*u;if(m<=0&&u-h>=0&&d-p>=0)return vu.subVectors(r,s),a=(u-h)/(u-h+(d-p)),t.copy(s).addScaledVector(vu,a);let g=1/(m+y+f);return o=y*g,a=f*g,t.copy(n).addScaledVector(Fs,o).addScaledVector(Bs,a)}equals(e){return e.a.equals(this.a)&&e.b.equals(this.b)&&e.c.equals(this.c)}},Xt=class{constructor(e=new B(1/0,1/0,1/0),t=new B(-1/0,-1/0,-1/0)){this.isBox3=!0,this.min=e,this.max=t}set(e,t){return this.min.copy(e),this.max.copy(t),this}setFromArray(e){this.makeEmpty();for(let t=0,n=e.length;t<n;t+=3)this.expandByPoint(Dn.fromArray(e,t));return this}setFromBufferAttribute(e){this.makeEmpty();for(let t=0,n=e.count;t<n;t++)this.expandByPoint(Dn.fromBufferAttribute(e,t));return this}setFromPoints(e){this.makeEmpty();for(let t=0,n=e.length;t<n;t++)this.expandByPoint(e[t]);return this}setFromCenterAndSize(e,t){let n=Dn.copy(t).multiplyScalar(.5);return this.min.copy(e).sub(n),this.max.copy(e).add(n),this}setFromObject(e,t=!1){return this.makeEmpty(),this.expandByObject(e,t)}clone(){return new this.constructor().copy(this)}copy(e){return this.min.copy(e.min),this.max.copy(e.max),this}makeEmpty(){return this.min.x=this.min.y=this.min.z=1/0,this.max.x=this.max.y=this.max.z=-1/0,this}isEmpty(){return this.max.x<this.min.x||this.max.y<this.min.y||this.max.z<this.min.z}getCenter(e){return this.isEmpty()?e.set(0,0,0):e.addVectors(this.min,this.max).multiplyScalar(.5)}getSize(e){return this.isEmpty()?e.set(0,0,0):e.subVectors(this.max,this.min)}expandByPoint(e){return this.min.min(e),this.max.max(e),this}expandByVector(e){return this.min.sub(e),this.max.add(e),this}expandByScalar(e){return this.min.addScalar(-e),this.max.addScalar(e),this}expandByObject(e,t=!1){e.updateWorldMatrix(!1,!1);let n=e.geometry;if(n!==void 0){let r=n.getAttribute("position");if(t===!0&&r!==void 0&&e.isInstancedMesh!==!0)for(let o=0,a=r.count;o<a;o++)e.isMesh===!0?e.getVertexPosition(o,Dn):Dn.fromBufferAttribute(r,o),Dn.applyMatrix4(e.matrixWorld),this.expandByPoint(Dn);else e.boundingBox!==void 0?(e.boundingBox===null&&e.computeBoundingBox(),No.copy(e.boundingBox)):(n.boundingBox===null&&n.computeBoundingBox(),No.copy(n.boundingBox)),No.applyMatrix4(e.matrixWorld),this.union(No)}let s=e.children;for(let r=0,o=s.length;r<o;r++)this.expandByObject(s[r],t);return this}containsPoint(e){return e.x>=this.min.x&&e.x<=this.max.x&&e.y>=this.min.y&&e.y<=this.max.y&&e.z>=this.min.z&&e.z<=this.max.z}containsBox(e){return this.min.x<=e.min.x&&e.max.x<=this.max.x&&this.min.y<=e.min.y&&e.max.y<=this.max.y&&this.min.z<=e.min.z&&e.max.z<=this.max.z}getParameter(e,t){return t.set((e.x-this.min.x)/(this.max.x-this.min.x),(e.y-this.min.y)/(this.max.y-this.min.y),(e.z-this.min.z)/(this.max.z-this.min.z))}intersectsBox(e){return e.max.x>=this.min.x&&e.min.x<=this.max.x&&e.max.y>=this.min.y&&e.min.y<=this.max.y&&e.max.z>=this.min.z&&e.min.z<=this.max.z}intersectsSphere(e){return this.clampPoint(e.center,Dn),Dn.distanceToSquared(e.center)<=e.radius*e.radius}intersectsPlane(e){let t,n;return e.normal.x>0?(t=e.normal.x*this.min.x,n=e.normal.x*this.max.x):(t=e.normal.x*this.max.x,n=e.normal.x*this.min.x),e.normal.y>0?(t+=e.normal.y*this.min.y,n+=e.normal.y*this.max.y):(t+=e.normal.y*this.max.y,n+=e.normal.y*this.min.y),e.normal.z>0?(t+=e.normal.z*this.min.z,n+=e.normal.z*this.max.z):(t+=e.normal.z*this.max.z,n+=e.normal.z*this.min.z),t<=-e.constant&&n>=-e.constant}intersectsTriangle(e){if(this.isEmpty())return!1;this.getCenter(Rr),Uo.subVectors(this.max,Rr),ks.subVectors(e.a,Rr),zs.subVectors(e.b,Rr),Hs.subVectors(e.c,Rr),Ni.subVectors(zs,ks),Ui.subVectors(Hs,zs),ss.subVectors(ks,Hs);let t=[0,-Ni.z,Ni.y,0,-Ui.z,Ui.y,0,-ss.z,ss.y,Ni.z,0,-Ni.x,Ui.z,0,-Ui.x,ss.z,0,-ss.x,-Ni.y,Ni.x,0,-Ui.y,Ui.x,0,-ss.y,ss.x,0];return!Qc(t,ks,zs,Hs,Uo)||(t=[1,0,0,0,1,0,0,0,1],!Qc(t,ks,zs,Hs,Uo))?!1:(Oo.crossVectors(Ni,Ui),t=[Oo.x,Oo.y,Oo.z],Qc(t,ks,zs,Hs,Uo))}clampPoint(e,t){return t.copy(e).clamp(this.min,this.max)}distanceToPoint(e){return this.clampPoint(e,Dn).distanceTo(e)}getBoundingSphere(e){return this.isEmpty()?e.makeEmpty():(this.getCenter(e.center),e.radius=this.getSize(Dn).length()*.5),e}intersect(e){return this.min.max(e.min),this.max.min(e.max),this.isEmpty()&&this.makeEmpty(),this}union(e){return this.min.min(e.min),this.max.max(e.max),this}applyMatrix4(e){return this.isEmpty()?this:(fi[0].set(this.min.x,this.min.y,this.min.z).applyMatrix4(e),fi[1].set(this.min.x,this.min.y,this.max.z).applyMatrix4(e),fi[2].set(this.min.x,this.max.y,this.min.z).applyMatrix4(e),fi[3].set(this.min.x,this.max.y,this.max.z).applyMatrix4(e),fi[4].set(this.max.x,this.min.y,this.min.z).applyMatrix4(e),fi[5].set(this.max.x,this.min.y,this.max.z).applyMatrix4(e),fi[6].set(this.max.x,this.max.y,this.min.z).applyMatrix4(e),fi[7].set(this.max.x,this.max.y,this.max.z).applyMatrix4(e),this.setFromPoints(fi),this)}translate(e){return this.min.add(e),this.max.add(e),this}equals(e){return e.min.equals(this.min)&&e.max.equals(this.max)}toJSON(){return{min:this.min.toArray(),max:this.max.toArray()}}fromJSON(e){return this.min.fromArray(e.min),this.max.fromArray(e.max),this}},fi=[new B,new B,new B,new B,new B,new B,new B,new B],Dn=new B,No=new Xt,ks=new B,zs=new B,Hs=new B,Ni=new B,Ui=new B,ss=new B,Rr=new B,Uo=new B,Oo=new B,rs=new B;function Qc(i,e,t,n,s){for(let r=0,o=i.length-3;r<=o;r+=3){rs.fromArray(i,r);let a=s.x*Math.abs(rs.x)+s.y*Math.abs(rs.y)+s.z*Math.abs(rs.z),c=e.dot(rs),l=t.dot(rs),h=n.dot(rs);if(Math.max(-Math.max(c,l,h),Math.min(c,l,h))>a)return!1}return!0}var Nt=new B,Fo=new de,Cp=0,yt=class extends Fn{constructor(e,t,n=!1){if(super(),Array.isArray(e))throw new TypeError("THREE.BufferAttribute: array should be a Typed Array.");this.isBufferAttribute=!0,Object.defineProperty(this,"id",{value:Cp++}),this.name="",this.array=e,this.itemSize=t,this.count=e!==void 0?e.length/t:0,this.normalized=n,this.usage=ma,this.updateRanges=[],this.gpuType=nn,this.version=0}onUploadCallback(){}set needsUpdate(e){e===!0&&this.version++}setUsage(e){return this.usage=e,this}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}copy(e){return this.name=e.name,this.array=new e.array.constructor(e.array),this.itemSize=e.itemSize,this.count=e.count,this.normalized=e.normalized,this.usage=e.usage,this.gpuType=e.gpuType,this}copyAt(e,t,n){e*=this.itemSize,n*=t.itemSize;for(let s=0,r=this.itemSize;s<r;s++)this.array[e+s]=t.array[n+s];return this}copyArray(e){return this.array.set(e),this}applyMatrix3(e){if(this.itemSize===2)for(let t=0,n=this.count;t<n;t++)Fo.fromBufferAttribute(this,t),Fo.applyMatrix3(e),this.setXY(t,Fo.x,Fo.y);else if(this.itemSize===3)for(let t=0,n=this.count;t<n;t++)Nt.fromBufferAttribute(this,t),Nt.applyMatrix3(e),this.setXYZ(t,Nt.x,Nt.y,Nt.z);return this}applyMatrix4(e){for(let t=0,n=this.count;t<n;t++)Nt.fromBufferAttribute(this,t),Nt.applyMatrix4(e),this.setXYZ(t,Nt.x,Nt.y,Nt.z);return this}applyNormalMatrix(e){for(let t=0,n=this.count;t<n;t++)Nt.fromBufferAttribute(this,t),Nt.applyNormalMatrix(e),this.setXYZ(t,Nt.x,Nt.y,Nt.z);return this}transformDirection(e){for(let t=0,n=this.count;t<n;t++)Nt.fromBufferAttribute(this,t),Nt.transformDirection(e),this.setXYZ(t,Nt.x,Nt.y,Nt.z);return this}set(e,t=0){return this.array.set(e,t),this}getComponent(e,t){let n=this.array[e*this.itemSize+t];return this.normalized&&(n=Nn(n,this.array)),n}setComponent(e,t,n){return this.normalized&&(n=ut(n,this.array)),this.array[e*this.itemSize+t]=n,this}getX(e){let t=this.array[e*this.itemSize];return this.normalized&&(t=Nn(t,this.array)),t}setX(e,t){return this.normalized&&(t=ut(t,this.array)),this.array[e*this.itemSize]=t,this}getY(e){let t=this.array[e*this.itemSize+1];return this.normalized&&(t=Nn(t,this.array)),t}setY(e,t){return this.normalized&&(t=ut(t,this.array)),this.array[e*this.itemSize+1]=t,this}getZ(e){let t=this.array[e*this.itemSize+2];return this.normalized&&(t=Nn(t,this.array)),t}setZ(e,t){return this.normalized&&(t=ut(t,this.array)),this.array[e*this.itemSize+2]=t,this}getW(e){let t=this.array[e*this.itemSize+3];return this.normalized&&(t=Nn(t,this.array)),t}setW(e,t){return this.normalized&&(t=ut(t,this.array)),this.array[e*this.itemSize+3]=t,this}setXY(e,t,n){return e*=this.itemSize,this.normalized&&(t=ut(t,this.array),n=ut(n,this.array)),this.array[e+0]=t,this.array[e+1]=n,this}setXYZ(e,t,n,s){return e*=this.itemSize,this.normalized&&(t=ut(t,this.array),n=ut(n,this.array),s=ut(s,this.array)),this.array[e+0]=t,this.array[e+1]=n,this.array[e+2]=s,this}setXYZW(e,t,n,s,r){return e*=this.itemSize,this.normalized&&(t=ut(t,this.array),n=ut(n,this.array),s=ut(s,this.array),r=ut(r,this.array)),this.array[e+0]=t,this.array[e+1]=n,this.array[e+2]=s,this.array[e+3]=r,this}onUpload(e){return this.onUploadCallback=e,this}clone(){return new this.constructor(this.array,this.itemSize).copy(this)}toJSON(){let e={itemSize:this.itemSize,type:this.array.constructor.name,array:Array.from(this.array),normalized:this.normalized};return this.name!==""&&(e.name=this.name),this.usage!==ma&&(e.usage=this.usage),e}dispose(){this.dispatchEvent({type:"dispose"})}};var Wr=class extends yt{constructor(e,t,n){super(new Uint16Array(e),t,n)}};var Xr=class extends yt{constructor(e,t,n){super(new Uint32Array(e),t,n)}};var Mt=class extends yt{constructor(e,t,n){super(new Float32Array(e),t,n)}},Pp=new Xt,Cr=new B,el=new B,rn=class{constructor(e=new B,t=-1){this.isSphere=!0,this.center=e,this.radius=t}set(e,t){return this.center.copy(e),this.radius=t,this}setFromPoints(e,t){let n=this.center;t!==void 0?n.copy(t):Pp.setFromPoints(e).getCenter(n);let s=0;for(let r=0,o=e.length;r<o;r++)s=Math.max(s,n.distanceToSquared(e[r]));return this.radius=Math.sqrt(s),this}copy(e){return this.center.copy(e.center),this.radius=e.radius,this}isEmpty(){return this.radius<0}makeEmpty(){return this.center.set(0,0,0),this.radius=-1,this}containsPoint(e){return e.distanceToSquared(this.center)<=this.radius*this.radius}distanceToPoint(e){return e.distanceTo(this.center)-this.radius}intersectsSphere(e){let t=this.radius+e.radius;return e.center.distanceToSquared(this.center)<=t*t}intersectsBox(e){return e.intersectsSphere(this)}intersectsPlane(e){return Math.abs(e.distanceToPoint(this.center))<=this.radius}clampPoint(e,t){let n=this.center.distanceToSquared(e);return t.copy(e),n>this.radius*this.radius&&(t.sub(this.center).normalize(),t.multiplyScalar(this.radius).add(this.center)),t}getBoundingBox(e){return this.isEmpty()?(e.makeEmpty(),e):(e.set(this.center,this.center),e.expandByScalar(this.radius),e)}applyMatrix4(e){return this.center.applyMatrix4(e),this.radius=this.radius*e.getMaxScaleOnAxis(),this}translate(e){return this.center.add(e),this}expandByPoint(e){if(this.isEmpty())return this.center.copy(e),this.radius=0,this;Cr.subVectors(e,this.center);let t=Cr.lengthSq();if(t>this.radius*this.radius){let n=Math.sqrt(t),s=(n-this.radius)*.5;this.center.addScaledVector(Cr,s/n),this.radius+=s}return this}union(e){return e.isEmpty()?this:this.isEmpty()?(this.copy(e),this):(this.center.equals(e.center)===!0?this.radius=Math.max(this.radius,e.radius):(el.subVectors(e.center,this.center).setLength(e.radius),this.expandByPoint(Cr.copy(e.center).add(el)),this.expandByPoint(Cr.copy(e.center).sub(el))),this)}equals(e){return e.center.equals(this.center)&&e.radius===this.radius}clone(){return new this.constructor().copy(this)}toJSON(){return{radius:this.radius,center:this.center.toArray()}}fromJSON(e){return this.radius=e.radius,this.center.fromArray(e.center),this}},Ip=0,Mn=new Je,tl=new At,Vs=new B,mn=new Xt,Pr=new Xt,Vt=new B,wt=class i extends Fn{constructor(){super(),this.isBufferGeometry=!0,Object.defineProperty(this,"id",{value:Ip++}),this.uuid=Tn(),this.name="",this.type="BufferGeometry",this.index=null,this.indirect=null,this.indirectOffset=0,this.attributes={},this.morphAttributes={},this.morphTargetsRelative=!1,this.groups=[],this.boundingBox=null,this.boundingSphere=null,this.drawRange={start:0,count:1/0},this.userData={},this._transformed=!1}getIndex(){return this.index}setIndex(e){return Array.isArray(e)?this.index=new(tp(e)?Xr:Wr)(e,1):this.index=e,this}setIndirect(e,t=0){return this.indirect=e,this.indirectOffset=t,this}getIndirect(){return this.indirect}getAttribute(e){return this.attributes[e]}setAttribute(e,t){return this.attributes[e]=t,this}deleteAttribute(e){return delete this.attributes[e],this}hasAttribute(e){return this.attributes[e]!==void 0}addGroup(e,t,n=0){this.groups.push({start:e,count:t,materialIndex:n})}clearGroups(){this.groups=[]}setDrawRange(e,t){this.drawRange.start=e,this.drawRange.count=t}applyMatrix4(e){let t=this.attributes.position;t!==void 0&&(t.applyMatrix4(e),t.needsUpdate=!0);let n=this.attributes.normal;if(n!==void 0){let r=new Ze().getNormalMatrix(e);n.applyNormalMatrix(r),n.needsUpdate=!0}let s=this.attributes.tangent;return s!==void 0&&(s.transformDirection(e),s.needsUpdate=!0),this.boundingBox!==null&&this.computeBoundingBox(),this.boundingSphere!==null&&this.computeBoundingSphere(),this._transformed=!0,this}applyQuaternion(e){return Mn.makeRotationFromQuaternion(e),this.applyMatrix4(Mn),this}rotateX(e){return Mn.makeRotationX(e),this.applyMatrix4(Mn),this}rotateY(e){return Mn.makeRotationY(e),this.applyMatrix4(Mn),this}rotateZ(e){return Mn.makeRotationZ(e),this.applyMatrix4(Mn),this}translate(e,t,n){return Mn.makeTranslation(e,t,n),this.applyMatrix4(Mn),this}scale(e,t,n){return Mn.makeScale(e,t,n),this.applyMatrix4(Mn),this}lookAt(e){return tl.lookAt(e),tl.updateMatrix(),this.applyMatrix4(tl.matrix),this}center(){return this.computeBoundingBox(),this.boundingBox.getCenter(Vs).negate(),this.translate(Vs.x,Vs.y,Vs.z),this}setFromPoints(e){let t=this.getAttribute("position");if(t===void 0){let n=[];for(let s=0,r=e.length;s<r;s++){let o=e[s];n.push(o.x,o.y,o.z||0)}this.setAttribute("position",new Mt(n,3))}else{let n=Math.min(e.length,t.count);for(let s=0;s<n;s++){let r=e[s];t.setXYZ(s,r.x,r.y,r.z||0)}e.length>t.count&&ke("BufferGeometry: Buffer size too small for points data. Use .dispose() and create a new geometry."),t.needsUpdate=!0}return this}computeBoundingBox(){this.boundingBox===null&&(this.boundingBox=new Xt);let e=this.attributes.position,t=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){Ke("BufferGeometry.computeBoundingBox(): GLBufferAttribute requires a manual bounding box.",this),this.boundingBox.set(new B(-1/0,-1/0,-1/0),new B(1/0,1/0,1/0));return}if(e!==void 0){if(this.boundingBox.setFromBufferAttribute(e),t)for(let n=0,s=t.length;n<s;n++){let r=t[n];mn.setFromBufferAttribute(r),this.morphTargetsRelative?(Vt.addVectors(this.boundingBox.min,mn.min),this.boundingBox.expandByPoint(Vt),Vt.addVectors(this.boundingBox.max,mn.max),this.boundingBox.expandByPoint(Vt)):(this.boundingBox.expandByPoint(mn.min),this.boundingBox.expandByPoint(mn.max))}}else this.boundingBox.makeEmpty();(isNaN(this.boundingBox.min.x)||isNaN(this.boundingBox.min.y)||isNaN(this.boundingBox.min.z))&&Ke('BufferGeometry.computeBoundingBox(): Computed min/max have NaN values. The "position" attribute is likely to have NaN values.',this)}computeBoundingSphere(){this.boundingSphere===null&&(this.boundingSphere=new rn);let e=this.attributes.position,t=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){Ke("BufferGeometry.computeBoundingSphere(): GLBufferAttribute requires a manual bounding sphere.",this),this.boundingSphere.set(new B,1/0);return}if(e){let n=this.boundingSphere.center;if(mn.setFromBufferAttribute(e),t)for(let r=0,o=t.length;r<o;r++){let a=t[r];Pr.setFromBufferAttribute(a),this.morphTargetsRelative?(Vt.addVectors(mn.min,Pr.min),mn.expandByPoint(Vt),Vt.addVectors(mn.max,Pr.max),mn.expandByPoint(Vt)):(mn.expandByPoint(Pr.min),mn.expandByPoint(Pr.max))}mn.getCenter(n);let s=0;for(let r=0,o=e.count;r<o;r++)Vt.fromBufferAttribute(e,r),s=Math.max(s,n.distanceToSquared(Vt));if(t)for(let r=0,o=t.length;r<o;r++){let a=t[r],c=this.morphTargetsRelative;for(let l=0,h=a.count;l<h;l++)Vt.fromBufferAttribute(a,l),c&&(Vs.fromBufferAttribute(e,l),Vt.add(Vs)),s=Math.max(s,n.distanceToSquared(Vt))}this.boundingSphere.radius=Math.sqrt(s),isNaN(this.boundingSphere.radius)&&Ke('BufferGeometry.computeBoundingSphere(): Computed radius is NaN. The "position" attribute is likely to have NaN values.',this)}}computeTangents(){let e=this.index,t=this.attributes;if(e===null||t.position===void 0||t.normal===void 0||t.uv===void 0){Ke("BufferGeometry: .computeTangents() failed. Missing required attributes (index, position, normal or uv)");return}let n=t.position,s=t.normal,r=t.uv,o=this.getAttribute("tangent");(o===void 0||o.count!==n.count)&&(o=new yt(new Float32Array(4*n.count),4),this.setAttribute("tangent",o));let a=[],c=[];for(let v=0;v<n.count;v++)a[v]=new B,c[v]=new B;let l=new B,h=new B,u=new B,f=new de,d=new de,p=new de,y=new B,m=new B;function g(v,R,F){l.fromBufferAttribute(n,v),h.fromBufferAttribute(n,R),u.fromBufferAttribute(n,F),f.fromBufferAttribute(r,v),d.fromBufferAttribute(r,R),p.fromBufferAttribute(r,F),h.sub(l),u.sub(l),d.sub(f),p.sub(f);let N=1/(d.x*p.y-p.x*d.y);isFinite(N)&&(y.copy(h).multiplyScalar(p.y).addScaledVector(u,-d.y).multiplyScalar(N),m.copy(u).multiplyScalar(d.x).addScaledVector(h,-p.x).multiplyScalar(N),a[v].add(y),a[R].add(y),a[F].add(y),c[v].add(m),c[R].add(m),c[F].add(m))}let E=this.groups;E.length===0&&(E=[{start:0,count:e.count}]);for(let v=0,R=E.length;v<R;++v){let F=E[v],N=F.start,H=F.count;for(let oe=N,ie=N+H;oe<ie;oe+=3)g(e.getX(oe+0),e.getX(oe+1),e.getX(oe+2))}let S=new B,x=new B,A=new B,w=new B;function I(v){A.fromBufferAttribute(s,v),w.copy(A);let R=a[v];S.copy(R),S.sub(A.multiplyScalar(A.dot(R))).normalize(),x.crossVectors(w,R);let N=x.dot(c[v])<0?-1:1;o.setXYZW(v,S.x,S.y,S.z,N)}for(let v=0,R=E.length;v<R;++v){let F=E[v],N=F.start,H=F.count;for(let oe=N,ie=N+H;oe<ie;oe+=3)I(e.getX(oe+0)),I(e.getX(oe+1)),I(e.getX(oe+2))}this._transformed=!0}computeVertexNormals(){let e=this.index,t=this.getAttribute("position");if(t!==void 0){let n=this.getAttribute("normal");if(n===void 0||n.count!==t.count)n=new yt(new Float32Array(t.count*3),3),this.setAttribute("normal",n);else for(let f=0,d=n.count;f<d;f++)n.setXYZ(f,0,0,0);let s=new B,r=new B,o=new B,a=new B,c=new B,l=new B,h=new B,u=new B;if(e)for(let f=0,d=e.count;f<d;f+=3){let p=e.getX(f+0),y=e.getX(f+1),m=e.getX(f+2);s.fromBufferAttribute(t,p),r.fromBufferAttribute(t,y),o.fromBufferAttribute(t,m),h.subVectors(o,r),u.subVectors(s,r),h.cross(u),a.fromBufferAttribute(n,p),c.fromBufferAttribute(n,y),l.fromBufferAttribute(n,m),a.add(h),c.add(h),l.add(h),n.setXYZ(p,a.x,a.y,a.z),n.setXYZ(y,c.x,c.y,c.z),n.setXYZ(m,l.x,l.y,l.z)}else for(let f=0,d=t.count;f<d;f+=3)s.fromBufferAttribute(t,f+0),r.fromBufferAttribute(t,f+1),o.fromBufferAttribute(t,f+2),h.subVectors(o,r),u.subVectors(s,r),h.cross(u),n.setXYZ(f+0,h.x,h.y,h.z),n.setXYZ(f+1,h.x,h.y,h.z),n.setXYZ(f+2,h.x,h.y,h.z);this.normalizeNormals(),n.needsUpdate=!0}}normalizeNormals(){let e=this.attributes.normal;for(let t=0,n=e.count;t<n;t++)Vt.fromBufferAttribute(e,t),Vt.normalize(),e.setXYZ(t,Vt.x,Vt.y,Vt.z)}toNonIndexed(){function e(a,c){let l=a.array,h=a.itemSize,u=a.normalized,f=new l.constructor(c.length*h),d=0,p=0;for(let y=0,m=c.length;y<m;y++){a.isInterleavedBufferAttribute?d=c[y]*a.data.stride+a.offset:d=c[y]*h;for(let g=0;g<h;g++)f[p++]=l[d++]}return new yt(f,h,u)}if(this.index===null)return ke("BufferGeometry.toNonIndexed(): BufferGeometry is already non-indexed."),this;let t=new i,n=this.index.array,s=this.attributes;for(let a in s){let c=s[a],l=e(c,n);t.setAttribute(a,l)}let r=this.morphAttributes;for(let a in r){let c=[],l=r[a];for(let h=0,u=l.length;h<u;h++){let f=l[h],d=e(f,n);c.push(d)}t.morphAttributes[a]=c}t.morphTargetsRelative=this.morphTargetsRelative;let o=this.groups;for(let a=0,c=o.length;a<c;a++){let l=o[a];t.addGroup(l.start,l.count,l.materialIndex)}return t}toJSON(){let e={metadata:{version:4.7,type:"BufferGeometry",generator:"BufferGeometry.toJSON"}};if(e.uuid=this.uuid,e.type=this.parameters!==void 0&&this._transformed===!0?"BufferGeometry":this.type,this.name!==""&&(e.name=this.name),Object.keys(this.userData).length>0&&(e.userData=this.userData),this.parameters!==void 0&&this._transformed!==!0){let c=this.parameters;for(let l in c)c[l]!==void 0&&(e[l]=c[l]);return e}e.data={attributes:{}};let t=this.index;t!==null&&(e.data.index={type:t.array.constructor.name,array:Array.prototype.slice.call(t.array)});let n=this.attributes;for(let c in n){let l=n[c];e.data.attributes[c]=l.toJSON(e.data)}let s={},r=!1;for(let c in this.morphAttributes){let l=this.morphAttributes[c],h=[];for(let u=0,f=l.length;u<f;u++){let d=l[u];h.push(d.toJSON(e.data))}h.length>0&&(s[c]=h,r=!0)}r&&(e.data.morphAttributes=s,e.data.morphTargetsRelative=this.morphTargetsRelative);let o=this.groups;o.length>0&&(e.data.groups=JSON.parse(JSON.stringify(o)));let a=this.boundingSphere;return a!==null&&(e.data.boundingSphere=a.toJSON()),e}clone(){return new this.constructor().copy(this)}copy(e){this.index=null,this.attributes={},this.morphAttributes={},this.groups=[],this.boundingBox=null,this.boundingSphere=null;let t={};this.name=e.name;let n=e.index;n!==null&&this.setIndex(n.clone());let s=e.attributes;for(let l in s){let h=s[l];this.setAttribute(l,h.clone(t))}let r=e.morphAttributes;for(let l in r){let h=[],u=r[l];for(let f=0,d=u.length;f<d;f++)h.push(u[f].clone(t));this.morphAttributes[l]=h}this.morphTargetsRelative=e.morphTargetsRelative;let o=e.groups;for(let l=0,h=o.length;l<h;l++){let u=o[l];this.addGroup(u.start,u.count,u.materialIndex)}let a=e.boundingBox;a!==null&&(this.boundingBox=a.clone());let c=e.boundingSphere;return c!==null&&(this.boundingSphere=c.clone()),this.drawRange.start=e.drawRange.start,this.drawRange.count=e.drawRange.count,this.userData=e.userData,this._transformed=e._transformed,this}dispose(){this.dispatchEvent({type:"dispose"})}},Hi=class{constructor(e,t){this.isInterleavedBuffer=!0,this.array=e,this.stride=t,this.count=e!==void 0?e.length/t:0,this.usage=ma,this.updateRanges=[],this.version=0,this.uuid=Tn()}onUploadCallback(){}set needsUpdate(e){e===!0&&this.version++}setUsage(e){return this.usage=e,this}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}copy(e){return this.array=new e.array.constructor(e.array),this.count=e.count,this.stride=e.stride,this.usage=e.usage,this}copyAt(e,t,n){e*=this.stride,n*=t.stride;for(let s=0,r=this.stride;s<r;s++)this.array[e+s]=t.array[n+s];return this}set(e,t=0){return this.array.set(e,t),this}clone(e){e.arrayBuffers===void 0&&(e.arrayBuffers={}),this.array.buffer._uuid===void 0&&(this.array.buffer._uuid=Tn()),e.arrayBuffers[this.array.buffer._uuid]===void 0&&(e.arrayBuffers[this.array.buffer._uuid]=this.array.slice(0).buffer);let t=new this.array.constructor(e.arrayBuffers[this.array.buffer._uuid]),n=new this.constructor(t,this.stride);return n.setUsage(this.usage),n}onUpload(e){return this.onUploadCallback=e,this}toJSON(e){return e.arrayBuffers===void 0&&(e.arrayBuffers={}),this.array.buffer._uuid===void 0&&(this.array.buffer._uuid=Tn()),e.arrayBuffers[this.array.buffer._uuid]===void 0&&(e.arrayBuffers[this.array.buffer._uuid]=Array.from(new Uint32Array(this.array.buffer))),{uuid:this.uuid,buffer:this.array.buffer._uuid,type:this.array.constructor.name,stride:this.stride}}},en=new B,Vi=class i{constructor(e,t,n,s=!1){this.isInterleavedBufferAttribute=!0,this.name="",this.data=e,this.itemSize=t,this.offset=n,this.normalized=s}get count(){return this.data.count}get array(){return this.data.array}set needsUpdate(e){this.data.needsUpdate=e}applyMatrix4(e){for(let t=0,n=this.data.count;t<n;t++)en.fromBufferAttribute(this,t),en.applyMatrix4(e),this.setXYZ(t,en.x,en.y,en.z);return this}applyNormalMatrix(e){for(let t=0,n=this.count;t<n;t++)en.fromBufferAttribute(this,t),en.applyNormalMatrix(e),this.setXYZ(t,en.x,en.y,en.z);return this}transformDirection(e){for(let t=0,n=this.count;t<n;t++)en.fromBufferAttribute(this,t),en.transformDirection(e),this.setXYZ(t,en.x,en.y,en.z);return this}getComponent(e,t){let n=this.array[e*this.data.stride+this.offset+t];return this.normalized&&(n=Nn(n,this.array)),n}setComponent(e,t,n){return this.normalized&&(n=ut(n,this.array)),this.data.array[e*this.data.stride+this.offset+t]=n,this}setX(e,t){return this.normalized&&(t=ut(t,this.array)),this.data.array[e*this.data.stride+this.offset]=t,this}setY(e,t){return this.normalized&&(t=ut(t,this.array)),this.data.array[e*this.data.stride+this.offset+1]=t,this}setZ(e,t){return this.normalized&&(t=ut(t,this.array)),this.data.array[e*this.data.stride+this.offset+2]=t,this}setW(e,t){return this.normalized&&(t=ut(t,this.array)),this.data.array[e*this.data.stride+this.offset+3]=t,this}getX(e){let t=this.data.array[e*this.data.stride+this.offset];return this.normalized&&(t=Nn(t,this.array)),t}getY(e){let t=this.data.array[e*this.data.stride+this.offset+1];return this.normalized&&(t=Nn(t,this.array)),t}getZ(e){let t=this.data.array[e*this.data.stride+this.offset+2];return this.normalized&&(t=Nn(t,this.array)),t}getW(e){let t=this.data.array[e*this.data.stride+this.offset+3];return this.normalized&&(t=Nn(t,this.array)),t}setXY(e,t,n){return e=e*this.data.stride+this.offset,this.normalized&&(t=ut(t,this.array),n=ut(n,this.array)),this.data.array[e+0]=t,this.data.array[e+1]=n,this}setXYZ(e,t,n,s){return e=e*this.data.stride+this.offset,this.normalized&&(t=ut(t,this.array),n=ut(n,this.array),s=ut(s,this.array)),this.data.array[e+0]=t,this.data.array[e+1]=n,this.data.array[e+2]=s,this}setXYZW(e,t,n,s,r){return e=e*this.data.stride+this.offset,this.normalized&&(t=ut(t,this.array),n=ut(n,this.array),s=ut(s,this.array),r=ut(r,this.array)),this.data.array[e+0]=t,this.data.array[e+1]=n,this.data.array[e+2]=s,this.data.array[e+3]=r,this}clone(e){if(e===void 0){Hr("InterleavedBufferAttribute.clone(): Cloning an interleaved buffer attribute will de-interleave buffer data.");let t=[];for(let n=0;n<this.count;n++){let s=n*this.data.stride+this.offset;for(let r=0;r<this.itemSize;r++)t.push(this.data.array[s+r])}return new yt(new this.array.constructor(t),this.itemSize,this.normalized)}else return e.interleavedBuffers===void 0&&(e.interleavedBuffers={}),e.interleavedBuffers[this.data.uuid]===void 0&&(e.interleavedBuffers[this.data.uuid]=this.data.clone(e)),new i(e.interleavedBuffers[this.data.uuid],this.itemSize,this.offset,this.normalized)}toJSON(e){if(e===void 0){Hr("InterleavedBufferAttribute.toJSON(): Serializing an interleaved buffer attribute will de-interleave buffer data.");let t=[];for(let n=0;n<this.count;n++){let s=n*this.data.stride+this.offset;for(let r=0;r<this.itemSize;r++)t.push(this.data.array[s+r])}return{itemSize:this.itemSize,type:this.array.constructor.name,array:t,normalized:this.normalized}}else return e.interleavedBuffers===void 0&&(e.interleavedBuffers={}),e.interleavedBuffers[this.data.uuid]===void 0&&(e.interleavedBuffers[this.data.uuid]=this.data.toJSON(e)),{isInterleavedBufferAttribute:!0,itemSize:this.itemSize,data:this.data.uuid,offset:this.offset,normalized:this.normalized}}},Lp=0,on=class extends Fn{constructor(){super(),this.isMaterial=!0,Object.defineProperty(this,"id",{value:Lp++}),this.uuid=Tn(),this.name="",this.type="Material",this.blending=hs,this.side=On,this.vertexColors=!1,this.opacity=1,this.transparent=!1,this.alphaHash=!1,this.blendSrc=oa,this.blendDst=aa,this.blendEquation=ki,this.blendSrcAlpha=null,this.blendDstAlpha=null,this.blendEquationAlpha=null,this.blendColor=new Ge(0,0,0),this.blendAlpha=0,this.depthFunc=us,this.depthTest=!0,this.depthWrite=!0,this.stencilWriteMask=255,this.stencilFunc=vl,this.stencilRef=0,this.stencilFuncMask=255,this.stencilFail=cs,this.stencilZFail=cs,this.stencilZPass=cs,this.stencilWrite=!1,this.clippingPlanes=null,this.clipIntersection=!1,this.clipShadows=!1,this.shadowSide=null,this.colorWrite=!0,this.precision=null,this.polygonOffset=!1,this.polygonOffsetFactor=0,this.polygonOffsetUnits=0,this.dithering=!1,this.alphaToCoverage=!1,this.premultipliedAlpha=!1,this.forceSinglePass=!1,this.allowOverride=!0,this.visible=!0,this.toneMapped=!0,this.userData={},this.version=0,this._alphaTest=0}get alphaTest(){return this._alphaTest}set alphaTest(e){this._alphaTest>0!=e>0&&this.version++,this._alphaTest=e}onBeforeRender(){}onBeforeCompile(){}customProgramCacheKey(){return this.onBeforeCompile.toString()}setValues(e){if(e!==void 0)for(let t in e){let n=e[t];if(n===void 0){ke(`Material: parameter '${t}' has value of undefined.`);continue}let s=this[t];if(s===void 0){ke(`Material: '${t}' is not a property of THREE.${this.type}.`);continue}s&&s.isColor?s.set(n):s&&s.isVector2&&n&&n.isVector2||s&&s.isEuler&&n&&n.isEuler||s&&s.isVector3&&n&&n.isVector3?s.copy(n):this[t]=n}}toJSON(e){let t=e===void 0||typeof e=="string";t&&(e={textures:{},images:{}});let n={metadata:{version:4.7,type:"Material",generator:"Material.toJSON"}};n.uuid=this.uuid,n.type=this.type,this.name!==""&&(n.name=this.name),this.color&&this.color.isColor&&(n.color=this.color.getHex()),this.roughness!==void 0&&(n.roughness=this.roughness),this.metalness!==void 0&&(n.metalness=this.metalness),this.sheen!==void 0&&(n.sheen=this.sheen),this.sheenColor&&this.sheenColor.isColor&&(n.sheenColor=this.sheenColor.getHex()),this.sheenRoughness!==void 0&&(n.sheenRoughness=this.sheenRoughness),this.emissive&&this.emissive.isColor&&(n.emissive=this.emissive.getHex()),this.emissiveIntensity!==void 0&&this.emissiveIntensity!==1&&(n.emissiveIntensity=this.emissiveIntensity),this.specular&&this.specular.isColor&&(n.specular=this.specular.getHex()),this.specularIntensity!==void 0&&(n.specularIntensity=this.specularIntensity),this.specularColor&&this.specularColor.isColor&&(n.specularColor=this.specularColor.getHex()),this.shininess!==void 0&&(n.shininess=this.shininess),this.clearcoat!==void 0&&(n.clearcoat=this.clearcoat),this.clearcoatRoughness!==void 0&&(n.clearcoatRoughness=this.clearcoatRoughness),this.clearcoatMap&&this.clearcoatMap.isTexture&&(n.clearcoatMap=this.clearcoatMap.toJSON(e).uuid),this.clearcoatRoughnessMap&&this.clearcoatRoughnessMap.isTexture&&(n.clearcoatRoughnessMap=this.clearcoatRoughnessMap.toJSON(e).uuid),this.clearcoatNormalMap&&this.clearcoatNormalMap.isTexture&&(n.clearcoatNormalMap=this.clearcoatNormalMap.toJSON(e).uuid,n.clearcoatNormalScale=this.clearcoatNormalScale.toArray()),this.sheenColorMap&&this.sheenColorMap.isTexture&&(n.sheenColorMap=this.sheenColorMap.toJSON(e).uuid),this.sheenRoughnessMap&&this.sheenRoughnessMap.isTexture&&(n.sheenRoughnessMap=this.sheenRoughnessMap.toJSON(e).uuid),this.dispersion!==void 0&&(n.dispersion=this.dispersion),this.iridescence!==void 0&&(n.iridescence=this.iridescence),this.iridescenceIOR!==void 0&&(n.iridescenceIOR=this.iridescenceIOR),this.iridescenceThicknessRange!==void 0&&(n.iridescenceThicknessRange=this.iridescenceThicknessRange),this.iridescenceMap&&this.iridescenceMap.isTexture&&(n.iridescenceMap=this.iridescenceMap.toJSON(e).uuid),this.iridescenceThicknessMap&&this.iridescenceThicknessMap.isTexture&&(n.iridescenceThicknessMap=this.iridescenceThicknessMap.toJSON(e).uuid),this.anisotropy!==void 0&&(n.anisotropy=this.anisotropy),this.anisotropyRotation!==void 0&&(n.anisotropyRotation=this.anisotropyRotation),this.anisotropyMap&&this.anisotropyMap.isTexture&&(n.anisotropyMap=this.anisotropyMap.toJSON(e).uuid),this.map&&this.map.isTexture&&(n.map=this.map.toJSON(e).uuid),this.matcap&&this.matcap.isTexture&&(n.matcap=this.matcap.toJSON(e).uuid),this.alphaMap&&this.alphaMap.isTexture&&(n.alphaMap=this.alphaMap.toJSON(e).uuid),this.lightMap&&this.lightMap.isTexture&&(n.lightMap=this.lightMap.toJSON(e).uuid,n.lightMapIntensity=this.lightMapIntensity),this.aoMap&&this.aoMap.isTexture&&(n.aoMap=this.aoMap.toJSON(e).uuid,n.aoMapIntensity=this.aoMapIntensity),this.bumpMap&&this.bumpMap.isTexture&&(n.bumpMap=this.bumpMap.toJSON(e).uuid,n.bumpScale=this.bumpScale),this.normalMap&&this.normalMap.isTexture&&(n.normalMap=this.normalMap.toJSON(e).uuid,n.normalMapType=this.normalMapType,n.normalScale=this.normalScale.toArray()),this.displacementMap&&this.displacementMap.isTexture&&(n.displacementMap=this.displacementMap.toJSON(e).uuid,n.displacementScale=this.displacementScale,n.displacementBias=this.displacementBias),this.roughnessMap&&this.roughnessMap.isTexture&&(n.roughnessMap=this.roughnessMap.toJSON(e).uuid),this.metalnessMap&&this.metalnessMap.isTexture&&(n.metalnessMap=this.metalnessMap.toJSON(e).uuid),this.emissiveMap&&this.emissiveMap.isTexture&&(n.emissiveMap=this.emissiveMap.toJSON(e).uuid),this.specularMap&&this.specularMap.isTexture&&(n.specularMap=this.specularMap.toJSON(e).uuid),this.specularIntensityMap&&this.specularIntensityMap.isTexture&&(n.specularIntensityMap=this.specularIntensityMap.toJSON(e).uuid),this.specularColorMap&&this.specularColorMap.isTexture&&(n.specularColorMap=this.specularColorMap.toJSON(e).uuid),this.envMap&&this.envMap.isTexture&&(n.envMap=this.envMap.toJSON(e).uuid,this.combine!==void 0&&(n.combine=this.combine)),this.envMapRotation!==void 0&&(n.envMapRotation=this.envMapRotation.toArray()),this.envMapIntensity!==void 0&&(n.envMapIntensity=this.envMapIntensity),this.reflectivity!==void 0&&(n.reflectivity=this.reflectivity),this.refractionRatio!==void 0&&(n.refractionRatio=this.refractionRatio),this.gradientMap&&this.gradientMap.isTexture&&(n.gradientMap=this.gradientMap.toJSON(e).uuid),this.transmission!==void 0&&(n.transmission=this.transmission),this.transmissionMap&&this.transmissionMap.isTexture&&(n.transmissionMap=this.transmissionMap.toJSON(e).uuid),this.thickness!==void 0&&(n.thickness=this.thickness),this.thicknessMap&&this.thicknessMap.isTexture&&(n.thicknessMap=this.thicknessMap.toJSON(e).uuid),this.attenuationDistance!==void 0&&this.attenuationDistance!==1/0&&(n.attenuationDistance=this.attenuationDistance),this.attenuationColor!==void 0&&(n.attenuationColor=this.attenuationColor.getHex()),this.size!==void 0&&(n.size=this.size),this.shadowSide!==null&&(n.shadowSide=this.shadowSide),this.sizeAttenuation!==void 0&&(n.sizeAttenuation=this.sizeAttenuation),this.blending!==hs&&(n.blending=this.blending),this.side!==On&&(n.side=this.side),this.vertexColors===!0&&(n.vertexColors=!0),this.opacity<1&&(n.opacity=this.opacity),this.transparent===!0&&(n.transparent=!0),this.blendSrc!==oa&&(n.blendSrc=this.blendSrc),this.blendDst!==aa&&(n.blendDst=this.blendDst),this.blendEquation!==ki&&(n.blendEquation=this.blendEquation),this.blendSrcAlpha!==null&&(n.blendSrcAlpha=this.blendSrcAlpha),this.blendDstAlpha!==null&&(n.blendDstAlpha=this.blendDstAlpha),this.blendEquationAlpha!==null&&(n.blendEquationAlpha=this.blendEquationAlpha),this.blendColor&&this.blendColor.isColor&&(n.blendColor=this.blendColor.getHex()),this.blendAlpha!==0&&(n.blendAlpha=this.blendAlpha),this.depthFunc!==us&&(n.depthFunc=this.depthFunc),this.depthTest===!1&&(n.depthTest=this.depthTest),this.depthWrite===!1&&(n.depthWrite=this.depthWrite),this.colorWrite===!1&&(n.colorWrite=this.colorWrite),this.stencilWriteMask!==255&&(n.stencilWriteMask=this.stencilWriteMask),this.stencilFunc!==vl&&(n.stencilFunc=this.stencilFunc),this.stencilRef!==0&&(n.stencilRef=this.stencilRef),this.stencilFuncMask!==255&&(n.stencilFuncMask=this.stencilFuncMask),this.stencilFail!==cs&&(n.stencilFail=this.stencilFail),this.stencilZFail!==cs&&(n.stencilZFail=this.stencilZFail),this.stencilZPass!==cs&&(n.stencilZPass=this.stencilZPass),this.stencilWrite===!0&&(n.stencilWrite=this.stencilWrite),this.rotation!==void 0&&this.rotation!==0&&(n.rotation=this.rotation),this.polygonOffset===!0&&(n.polygonOffset=!0),this.polygonOffsetFactor!==0&&(n.polygonOffsetFactor=this.polygonOffsetFactor),this.polygonOffsetUnits!==0&&(n.polygonOffsetUnits=this.polygonOffsetUnits),this.linewidth!==void 0&&this.linewidth!==1&&(n.linewidth=this.linewidth),this.dashSize!==void 0&&(n.dashSize=this.dashSize),this.gapSize!==void 0&&(n.gapSize=this.gapSize),this.scale!==void 0&&(n.scale=this.scale),this.dithering===!0&&(n.dithering=!0),this.alphaTest>0&&(n.alphaTest=this.alphaTest),this.alphaHash===!0&&(n.alphaHash=!0),this.alphaToCoverage===!0&&(n.alphaToCoverage=!0),this.premultipliedAlpha===!0&&(n.premultipliedAlpha=!0),this.forceSinglePass===!0&&(n.forceSinglePass=!0),this.allowOverride===!1&&(n.allowOverride=!1),this.wireframe===!0&&(n.wireframe=!0),this.wireframeLinewidth>1&&(n.wireframeLinewidth=this.wireframeLinewidth),this.wireframeLinecap!=="round"&&(n.wireframeLinecap=this.wireframeLinecap),this.wireframeLinejoin!=="round"&&(n.wireframeLinejoin=this.wireframeLinejoin),this.flatShading===!0&&(n.flatShading=!0),this.visible===!1&&(n.visible=!1),this.toneMapped===!1&&(n.toneMapped=!1),this.fog===!1&&(n.fog=!1),Object.keys(this.userData).length>0&&(n.userData=this.userData);function s(r){let o=[];for(let a in r){let c=r[a];delete c.metadata,o.push(c)}return o}if(t){let r=s(e.textures),o=s(e.images);r.length>0&&(n.textures=r),o.length>0&&(n.images=o)}return n}fromJSON(e,t){if(e.uuid!==void 0&&(this.uuid=e.uuid),e.name!==void 0&&(this.name=e.name),e.color!==void 0&&this.color!==void 0&&this.color.setHex(e.color),e.roughness!==void 0&&(this.roughness=e.roughness),e.metalness!==void 0&&(this.metalness=e.metalness),e.sheen!==void 0&&(this.sheen=e.sheen),e.sheenColor!==void 0&&(this.sheenColor=new Ge().setHex(e.sheenColor)),e.sheenRoughness!==void 0&&(this.sheenRoughness=e.sheenRoughness),e.emissive!==void 0&&this.emissive!==void 0&&this.emissive.setHex(e.emissive),e.specular!==void 0&&this.specular!==void 0&&this.specular.setHex(e.specular),e.specularIntensity!==void 0&&(this.specularIntensity=e.specularIntensity),e.specularColor!==void 0&&this.specularColor!==void 0&&this.specularColor.setHex(e.specularColor),e.shininess!==void 0&&(this.shininess=e.shininess),e.clearcoat!==void 0&&(this.clearcoat=e.clearcoat),e.clearcoatRoughness!==void 0&&(this.clearcoatRoughness=e.clearcoatRoughness),e.dispersion!==void 0&&(this.dispersion=e.dispersion),e.iridescence!==void 0&&(this.iridescence=e.iridescence),e.iridescenceIOR!==void 0&&(this.iridescenceIOR=e.iridescenceIOR),e.iridescenceThicknessRange!==void 0&&(this.iridescenceThicknessRange=e.iridescenceThicknessRange),e.transmission!==void 0&&(this.transmission=e.transmission),e.thickness!==void 0&&(this.thickness=e.thickness),e.attenuationDistance!==void 0&&(this.attenuationDistance=e.attenuationDistance),e.attenuationColor!==void 0&&this.attenuationColor!==void 0&&this.attenuationColor.setHex(e.attenuationColor),e.anisotropy!==void 0&&(this.anisotropy=e.anisotropy),e.anisotropyRotation!==void 0&&(this.anisotropyRotation=e.anisotropyRotation),e.fog!==void 0&&(this.fog=e.fog),e.flatShading!==void 0&&(this.flatShading=e.flatShading),e.blending!==void 0&&(this.blending=e.blending),e.combine!==void 0&&(this.combine=e.combine),e.side!==void 0&&(this.side=e.side),e.shadowSide!==void 0&&(this.shadowSide=e.shadowSide),e.opacity!==void 0&&(this.opacity=e.opacity),e.transparent!==void 0&&(this.transparent=e.transparent),e.alphaTest!==void 0&&(this.alphaTest=e.alphaTest),e.alphaHash!==void 0&&(this.alphaHash=e.alphaHash),e.depthFunc!==void 0&&(this.depthFunc=e.depthFunc),e.depthTest!==void 0&&(this.depthTest=e.depthTest),e.depthWrite!==void 0&&(this.depthWrite=e.depthWrite),e.colorWrite!==void 0&&(this.colorWrite=e.colorWrite),e.blendSrc!==void 0&&(this.blendSrc=e.blendSrc),e.blendDst!==void 0&&(this.blendDst=e.blendDst),e.blendEquation!==void 0&&(this.blendEquation=e.blendEquation),e.blendSrcAlpha!==void 0&&(this.blendSrcAlpha=e.blendSrcAlpha),e.blendDstAlpha!==void 0&&(this.blendDstAlpha=e.blendDstAlpha),e.blendEquationAlpha!==void 0&&(this.blendEquationAlpha=e.blendEquationAlpha),e.blendColor!==void 0&&this.blendColor!==void 0&&this.blendColor.setHex(e.blendColor),e.blendAlpha!==void 0&&(this.blendAlpha=e.blendAlpha),e.stencilWriteMask!==void 0&&(this.stencilWriteMask=e.stencilWriteMask),e.stencilFunc!==void 0&&(this.stencilFunc=e.stencilFunc),e.stencilRef!==void 0&&(this.stencilRef=e.stencilRef),e.stencilFuncMask!==void 0&&(this.stencilFuncMask=e.stencilFuncMask),e.stencilFail!==void 0&&(this.stencilFail=e.stencilFail),e.stencilZFail!==void 0&&(this.stencilZFail=e.stencilZFail),e.stencilZPass!==void 0&&(this.stencilZPass=e.stencilZPass),e.stencilWrite!==void 0&&(this.stencilWrite=e.stencilWrite),e.wireframe!==void 0&&(this.wireframe=e.wireframe),e.wireframeLinewidth!==void 0&&(this.wireframeLinewidth=e.wireframeLinewidth),e.wireframeLinecap!==void 0&&(this.wireframeLinecap=e.wireframeLinecap),e.wireframeLinejoin!==void 0&&(this.wireframeLinejoin=e.wireframeLinejoin),e.rotation!==void 0&&(this.rotation=e.rotation),e.linewidth!==void 0&&(this.linewidth=e.linewidth),e.dashSize!==void 0&&(this.dashSize=e.dashSize),e.gapSize!==void 0&&(this.gapSize=e.gapSize),e.scale!==void 0&&(this.scale=e.scale),e.polygonOffset!==void 0&&(this.polygonOffset=e.polygonOffset),e.polygonOffsetFactor!==void 0&&(this.polygonOffsetFactor=e.polygonOffsetFactor),e.polygonOffsetUnits!==void 0&&(this.polygonOffsetUnits=e.polygonOffsetUnits),e.dithering!==void 0&&(this.dithering=e.dithering),e.alphaToCoverage!==void 0&&(this.alphaToCoverage=e.alphaToCoverage),e.premultipliedAlpha!==void 0&&(this.premultipliedAlpha=e.premultipliedAlpha),e.forceSinglePass!==void 0&&(this.forceSinglePass=e.forceSinglePass),e.allowOverride!==void 0&&(this.allowOverride=e.allowOverride),e.visible!==void 0&&(this.visible=e.visible),e.toneMapped!==void 0&&(this.toneMapped=e.toneMapped),e.userData!==void 0&&(this.userData=e.userData),e.vertexColors!==void 0&&(typeof e.vertexColors=="number"?this.vertexColors=e.vertexColors>0:this.vertexColors=e.vertexColors),e.size!==void 0&&(this.size=e.size),e.sizeAttenuation!==void 0&&(this.sizeAttenuation=e.sizeAttenuation),e.map!==void 0&&(this.map=t[e.map]||null),e.matcap!==void 0&&(this.matcap=t[e.matcap]||null),e.alphaMap!==void 0&&(this.alphaMap=t[e.alphaMap]||null),e.bumpMap!==void 0&&(this.bumpMap=t[e.bumpMap]||null),e.bumpScale!==void 0&&(this.bumpScale=e.bumpScale),e.normalMap!==void 0&&(this.normalMap=t[e.normalMap]||null),e.normalMapType!==void 0&&(this.normalMapType=e.normalMapType),e.normalScale!==void 0){let n=e.normalScale;Array.isArray(n)===!1&&(n=[n,n]),this.normalScale=new de().fromArray(n)}return e.displacementMap!==void 0&&(this.displacementMap=t[e.displacementMap]||null),e.displacementScale!==void 0&&(this.displacementScale=e.displacementScale),e.displacementBias!==void 0&&(this.displacementBias=e.displacementBias),e.roughnessMap!==void 0&&(this.roughnessMap=t[e.roughnessMap]||null),e.metalnessMap!==void 0&&(this.metalnessMap=t[e.metalnessMap]||null),e.emissiveMap!==void 0&&(this.emissiveMap=t[e.emissiveMap]||null),e.emissiveIntensity!==void 0&&(this.emissiveIntensity=e.emissiveIntensity),e.specularMap!==void 0&&(this.specularMap=t[e.specularMap]||null),e.specularIntensityMap!==void 0&&(this.specularIntensityMap=t[e.specularIntensityMap]||null),e.specularColorMap!==void 0&&(this.specularColorMap=t[e.specularColorMap]||null),e.envMap!==void 0&&(this.envMap=t[e.envMap]||null),e.envMapRotation!==void 0&&this.envMapRotation.fromArray(e.envMapRotation),e.envMapIntensity!==void 0&&(this.envMapIntensity=e.envMapIntensity),e.reflectivity!==void 0&&(this.reflectivity=e.reflectivity),e.refractionRatio!==void 0&&(this.refractionRatio=e.refractionRatio),e.lightMap!==void 0&&(this.lightMap=t[e.lightMap]||null),e.lightMapIntensity!==void 0&&(this.lightMapIntensity=e.lightMapIntensity),e.aoMap!==void 0&&(this.aoMap=t[e.aoMap]||null),e.aoMapIntensity!==void 0&&(this.aoMapIntensity=e.aoMapIntensity),e.gradientMap!==void 0&&(this.gradientMap=t[e.gradientMap]||null),e.clearcoatMap!==void 0&&(this.clearcoatMap=t[e.clearcoatMap]||null),e.clearcoatRoughnessMap!==void 0&&(this.clearcoatRoughnessMap=t[e.clearcoatRoughnessMap]||null),e.clearcoatNormalMap!==void 0&&(this.clearcoatNormalMap=t[e.clearcoatNormalMap]||null),e.clearcoatNormalScale!==void 0&&(this.clearcoatNormalScale=new de().fromArray(e.clearcoatNormalScale)),e.iridescenceMap!==void 0&&(this.iridescenceMap=t[e.iridescenceMap]||null),e.iridescenceThicknessMap!==void 0&&(this.iridescenceThicknessMap=t[e.iridescenceThicknessMap]||null),e.transmissionMap!==void 0&&(this.transmissionMap=t[e.transmissionMap]||null),e.thicknessMap!==void 0&&(this.thicknessMap=t[e.thicknessMap]||null),e.anisotropyMap!==void 0&&(this.anisotropyMap=t[e.anisotropyMap]||null),e.sheenColorMap!==void 0&&(this.sheenColorMap=t[e.sheenColorMap]||null),e.sheenRoughnessMap!==void 0&&(this.sheenRoughnessMap=t[e.sheenRoughnessMap]||null),this}clone(){return new this.constructor().copy(this)}copy(e){this.name=e.name,this.blending=e.blending,this.side=e.side,this.vertexColors=e.vertexColors,this.opacity=e.opacity,this.transparent=e.transparent,this.blendSrc=e.blendSrc,this.blendDst=e.blendDst,this.blendEquation=e.blendEquation,this.blendSrcAlpha=e.blendSrcAlpha,this.blendDstAlpha=e.blendDstAlpha,this.blendEquationAlpha=e.blendEquationAlpha,this.blendColor.copy(e.blendColor),this.blendAlpha=e.blendAlpha,this.depthFunc=e.depthFunc,this.depthTest=e.depthTest,this.depthWrite=e.depthWrite,this.stencilWriteMask=e.stencilWriteMask,this.stencilFunc=e.stencilFunc,this.stencilRef=e.stencilRef,this.stencilFuncMask=e.stencilFuncMask,this.stencilFail=e.stencilFail,this.stencilZFail=e.stencilZFail,this.stencilZPass=e.stencilZPass,this.stencilWrite=e.stencilWrite;let t=e.clippingPlanes,n=null;if(t!==null){let s=t.length;n=new Array(s);for(let r=0;r!==s;++r)n[r]=t[r].clone()}return this.clippingPlanes=n,this.clipIntersection=e.clipIntersection,this.clipShadows=e.clipShadows,this.shadowSide=e.shadowSide,this.colorWrite=e.colorWrite,this.precision=e.precision,this.polygonOffset=e.polygonOffset,this.polygonOffsetFactor=e.polygonOffsetFactor,this.polygonOffsetUnits=e.polygonOffsetUnits,this.dithering=e.dithering,this.alphaTest=e.alphaTest,this.alphaHash=e.alphaHash,this.alphaToCoverage=e.alphaToCoverage,this.premultipliedAlpha=e.premultipliedAlpha,this.forceSinglePass=e.forceSinglePass,this.allowOverride=e.allowOverride,this.visible=e.visible,this.toneMapped=e.toneMapped,this.userData=JSON.parse(JSON.stringify(e.userData)),this}dispose(){this.dispatchEvent({type:"dispose"})}set needsUpdate(e){e===!0&&this.version++}};var di=new B,nl=new B,Bo=new B,Oi=new B,il=new B,ko=new B,sl=new B,_i=class{constructor(e=new B,t=new B(0,0,-1)){this.origin=e,this.direction=t}set(e,t){return this.origin.copy(e),this.direction.copy(t),this}copy(e){return this.origin.copy(e.origin),this.direction.copy(e.direction),this}at(e,t){return t.copy(this.origin).addScaledVector(this.direction,e)}lookAt(e){return this.direction.copy(e).sub(this.origin).normalize(),this}recast(e){return this.origin.copy(this.at(e,di)),this}closestPointToPoint(e,t){t.subVectors(e,this.origin);let n=t.dot(this.direction);return n<0?t.copy(this.origin):t.copy(this.origin).addScaledVector(this.direction,n)}distanceToPoint(e){return Math.sqrt(this.distanceSqToPoint(e))}distanceSqToPoint(e){let t=di.subVectors(e,this.origin).dot(this.direction);return t<0?this.origin.distanceToSquared(e):(di.copy(this.origin).addScaledVector(this.direction,t),di.distanceToSquared(e))}distanceSqToSegment(e,t,n,s){nl.copy(e).add(t).multiplyScalar(.5),Bo.copy(t).sub(e).normalize(),Oi.copy(this.origin).sub(nl);let r=e.distanceTo(t)*.5,o=-this.direction.dot(Bo),a=Oi.dot(this.direction),c=-Oi.dot(Bo),l=Oi.lengthSq(),h=Math.abs(1-o*o),u,f,d,p;if(h>0)if(u=o*c-a,f=o*a-c,p=r*h,u>=0)if(f>=-p)if(f<=p){let y=1/h;u*=y,f*=y,d=u*(u+o*f+2*a)+f*(o*u+f+2*c)+l}else f=r,u=Math.max(0,-(o*f+a)),d=-u*u+f*(f+2*c)+l;else f=-r,u=Math.max(0,-(o*f+a)),d=-u*u+f*(f+2*c)+l;else f<=-p?(u=Math.max(0,-(-o*r+a)),f=u>0?-r:Math.min(Math.max(-r,-c),r),d=-u*u+f*(f+2*c)+l):f<=p?(u=0,f=Math.min(Math.max(-r,-c),r),d=f*(f+2*c)+l):(u=Math.max(0,-(o*r+a)),f=u>0?r:Math.min(Math.max(-r,-c),r),d=-u*u+f*(f+2*c)+l);else f=o>0?-r:r,u=Math.max(0,-(o*f+a)),d=-u*u+f*(f+2*c)+l;return n&&n.copy(this.origin).addScaledVector(this.direction,u),s&&s.copy(nl).addScaledVector(Bo,f),d}intersectSphere(e,t){di.subVectors(e.center,this.origin);let n=di.dot(this.direction),s=di.dot(di)-n*n,r=e.radius*e.radius;if(s>r)return null;let o=Math.sqrt(r-s),a=n-o,c=n+o;return c<0?null:a<0?this.at(c,t):this.at(a,t)}intersectsSphere(e){return e.radius<0?!1:this.distanceSqToPoint(e.center)<=e.radius*e.radius}distanceToPlane(e){let t=e.normal.dot(this.direction);if(t===0)return e.distanceToPoint(this.origin)===0?0:null;let n=-(this.origin.dot(e.normal)+e.constant)/t;return n>=0?n:null}intersectPlane(e,t){let n=this.distanceToPlane(e);return n===null?null:this.at(n,t)}intersectsPlane(e){let t=e.distanceToPoint(this.origin);return t===0||e.normal.dot(this.direction)*t<0}intersectBox(e,t){let n,s,r,o,a,c,l=1/this.direction.x,h=1/this.direction.y,u=1/this.direction.z,f=this.origin;return l>=0?(n=(e.min.x-f.x)*l,s=(e.max.x-f.x)*l):(n=(e.max.x-f.x)*l,s=(e.min.x-f.x)*l),h>=0?(r=(e.min.y-f.y)*h,o=(e.max.y-f.y)*h):(r=(e.max.y-f.y)*h,o=(e.min.y-f.y)*h),n>o||r>s||((r>n||isNaN(n))&&(n=r),(o<s||isNaN(s))&&(s=o),u>=0?(a=(e.min.z-f.z)*u,c=(e.max.z-f.z)*u):(a=(e.max.z-f.z)*u,c=(e.min.z-f.z)*u),n>c||a>s)||((a>n||n!==n)&&(n=a),(c<s||s!==s)&&(s=c),s<0)?null:this.at(n>=0?n:s,t)}intersectsBox(e){return this.intersectBox(e,di)!==null}intersectTriangle(e,t,n,s,r){il.subVectors(t,e),ko.subVectors(n,e),sl.crossVectors(il,ko);let o=this.direction.dot(sl),a;if(o>0){if(s)return null;a=1}else if(o<0)a=-1,o=-o;else return null;Oi.subVectors(this.origin,e);let c=a*this.direction.dot(ko.crossVectors(Oi,ko));if(c<0)return null;let l=a*this.direction.dot(il.cross(Oi));if(l<0||c+l>o)return null;let h=-a*Oi.dot(sl);return h<0?null:this.at(h/o,r)}applyMatrix4(e){return this.origin.applyMatrix4(e),this.direction.transformDirection(e),this}equals(e){return e.origin.equals(this.origin)&&e.direction.equals(this.direction)}clone(){return new this.constructor().copy(this)}},Ot=class extends on{constructor(e){super(),this.isMeshBasicMaterial=!0,this.type="MeshBasicMaterial",this.color=new Ge(16777215),this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.specularMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new gi,this.combine=Ol,this.reflectivity=1,this.refractionRatio=.98,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.specularMap=e.specularMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.combine=e.combine,this.reflectivity=e.reflectivity,this.refractionRatio=e.refractionRatio,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.fog=e.fog,this}},bu=new Je,os=new _i,zo=new rn,Mu=new B,Ho=new B,Vo=new B,Go=new B,rl=new B,Wo=new B,Su=new B,Xo=new B,ht=class extends At{constructor(e=new wt,t=new Ot){super(),this.isMesh=!0,this.type="Mesh",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.count=1,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),e.morphTargetInfluences!==void 0&&(this.morphTargetInfluences=e.morphTargetInfluences.slice()),e.morphTargetDictionary!==void 0&&(this.morphTargetDictionary=Object.assign({},e.morphTargetDictionary)),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}updateMorphTargets(){let t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){let s=t[n[0]];if(s!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let r=0,o=s.length;r<o;r++){let a=s[r].name||String(r);this.morphTargetInfluences.push(0),this.morphTargetDictionary[a]=r}}}}getVertexPosition(e,t){let n=this.geometry,s=n.attributes.position,r=n.morphAttributes.position,o=n.morphTargetsRelative;t.fromBufferAttribute(s,e);let a=this.morphTargetInfluences;if(r&&a){Wo.set(0,0,0);for(let c=0,l=r.length;c<l;c++){let h=a[c],u=r[c];h!==0&&(rl.fromBufferAttribute(u,e),o?Wo.addScaledVector(rl,h):Wo.addScaledVector(rl.sub(t),h))}t.add(Wo)}return t}raycast(e,t){let n=this.geometry,s=this.material,r=this.matrixWorld;s!==void 0&&(n.boundingSphere===null&&n.computeBoundingSphere(),zo.copy(n.boundingSphere),zo.applyMatrix4(r),os.copy(e.ray).recast(e.near),!(zo.containsPoint(os.origin)===!1&&(os.intersectSphere(zo,Mu)===null||os.origin.distanceToSquared(Mu)>(e.far-e.near)**2))&&(bu.copy(r).invert(),os.copy(e.ray).applyMatrix4(bu),!(n.boundingBox!==null&&os.intersectsBox(n.boundingBox)===!1)&&this._computeIntersections(e,t,os)))}_computeIntersections(e,t,n){let s,r=this.geometry,o=this.material,a=r.index,c=r.attributes.position,l=r.attributes.uv,h=r.attributes.uv1,u=r.attributes.normal,f=r.groups,d=r.drawRange;if(a!==null)if(Array.isArray(o))for(let p=0,y=f.length;p<y;p++){let m=f[p],g=o[m.materialIndex],E=Math.max(m.start,d.start),S=Math.min(a.count,Math.min(m.start+m.count,d.start+d.count));for(let x=E,A=S;x<A;x+=3){let w=a.getX(x),I=a.getX(x+1),v=a.getX(x+2);s=qo(this,g,e,n,l,h,u,w,I,v),s&&(s.faceIndex=Math.floor(x/3),s.face.materialIndex=m.materialIndex,t.push(s))}}else{let p=Math.max(0,d.start),y=Math.min(a.count,d.start+d.count);for(let m=p,g=y;m<g;m+=3){let E=a.getX(m),S=a.getX(m+1),x=a.getX(m+2);s=qo(this,o,e,n,l,h,u,E,S,x),s&&(s.faceIndex=Math.floor(m/3),t.push(s))}}else if(c!==void 0)if(Array.isArray(o))for(let p=0,y=f.length;p<y;p++){let m=f[p],g=o[m.materialIndex],E=Math.max(m.start,d.start),S=Math.min(c.count,Math.min(m.start+m.count,d.start+d.count));for(let x=E,A=S;x<A;x+=3){let w=x,I=x+1,v=x+2;s=qo(this,g,e,n,l,h,u,w,I,v),s&&(s.faceIndex=Math.floor(x/3),s.face.materialIndex=m.materialIndex,t.push(s))}}else{let p=Math.max(0,d.start),y=Math.min(c.count,d.start+d.count);for(let m=p,g=y;m<g;m+=3){let E=m,S=m+1,x=m+2;s=qo(this,o,e,n,l,h,u,E,S,x),s&&(s.faceIndex=Math.floor(m/3),t.push(s))}}}};function Dp(i,e,t,n,s,r,o,a){let c;if(e.side===Ft?c=n.intersectTriangle(o,r,s,!0,a):c=n.intersectTriangle(s,r,o,e.side===On,a),c===null)return null;Xo.copy(a),Xo.applyMatrix4(i.matrixWorld);let l=t.ray.origin.distanceTo(Xo);return l<t.near||l>t.far?null:{distance:l,point:Xo.clone(),object:i}}function qo(i,e,t,n,s,r,o,a,c,l){i.getVertexPosition(a,Ho),i.getVertexPosition(c,Vo),i.getVertexPosition(l,Go);let h=Dp(i,e,t,n,Ho,Vo,Go,Su);if(h){let u=new B;Bi.getBarycoord(Su,Ho,Vo,Go,u),s&&(h.uv=Bi.getInterpolatedAttribute(s,a,c,l,u,new de)),r&&(h.uv1=Bi.getInterpolatedAttribute(r,a,c,l,u,new de)),o&&(h.normal=Bi.getInterpolatedAttribute(o,a,c,l,u,new B),h.normal.dot(n.direction)>0&&h.normal.multiplyScalar(-1));let f={a,b:c,c:l,normal:new B,materialIndex:0};Bi.getNormal(Ho,Vo,Go,f.normal),h.face=f,h.barycoord=u}return h}var Ir=new ft,Tu=new ft,Eu=new ft,Np=new ft,Au=new Je,Yo=new B,ol=new rn,wu=new Je,al=new _i,qr=class extends ht{constructor(e,t){super(e,t),this.isSkinnedMesh=!0,this.type="SkinnedMesh",this.bindMode=ml,this.bindMatrix=new Je,this.bindMatrixInverse=new Je,this.boundingBox=null,this.boundingSphere=null}computeBoundingBox(){let e=this.geometry;this.boundingBox===null&&(this.boundingBox=new Xt),this.boundingBox.makeEmpty();let t=e.getAttribute("position");for(let n=0;n<t.count;n++)this.getVertexPosition(n,Yo),this.boundingBox.expandByPoint(Yo)}computeBoundingSphere(){let e=this.geometry;this.boundingSphere===null&&(this.boundingSphere=new rn),this.boundingSphere.makeEmpty();let t=e.getAttribute("position");for(let n=0;n<t.count;n++)this.getVertexPosition(n,Yo),this.boundingSphere.expandByPoint(Yo)}copy(e,t){return super.copy(e,t),this.bindMode=e.bindMode,this.bindMatrix.copy(e.bindMatrix),this.bindMatrixInverse.copy(e.bindMatrixInverse),this.skeleton=e.skeleton,e.boundingBox!==null&&(this.boundingBox=e.boundingBox.clone()),e.boundingSphere!==null&&(this.boundingSphere=e.boundingSphere.clone()),this}raycast(e,t){let n=this.material,s=this.matrixWorld;n!==void 0&&(this.boundingSphere===null&&this.computeBoundingSphere(),ol.copy(this.boundingSphere),ol.applyMatrix4(s),e.ray.intersectsSphere(ol)!==!1&&(wu.copy(s).invert(),al.copy(e.ray).applyMatrix4(wu),!(this.boundingBox!==null&&al.intersectsBox(this.boundingBox)===!1)&&this._computeIntersections(e,t,al)))}getVertexPosition(e,t){return super.getVertexPosition(e,t),this.applyBoneTransform(e,t),t}bind(e,t){this.skeleton=e,t===void 0&&(this.updateMatrixWorld(!0),this.skeleton.calculateInverses(),t=this.matrixWorld),this.bindMatrix.copy(t),this.bindMatrixInverse.copy(t).invert()}pose(){this.skeleton.pose()}normalizeSkinWeights(){let e=new ft,t=this.geometry.attributes.skinWeight;for(let n=0,s=t.count;n<s;n++){e.fromBufferAttribute(t,n);let r=1/e.manhattanLength();r!==1/0?e.multiplyScalar(r):e.set(1,0,0,0),t.setXYZW(n,e.x,e.y,e.z,e.w)}}updateMatrixWorld(e){super.updateMatrixWorld(e),this.bindMode===ml?this.bindMatrixInverse.copy(this.matrixWorld).invert():this.bindMode===bf?this.bindMatrixInverse.copy(this.bindMatrix).invert():ke("SkinnedMesh: Unrecognized bindMode: "+this.bindMode)}applyBoneTransform(e,t){let n=this.skeleton,s=this.geometry;Tu.fromBufferAttribute(s.attributes.skinIndex,e),Eu.fromBufferAttribute(s.attributes.skinWeight,e),t.isVector4?(Ir.copy(t),t.set(0,0,0,0)):(Ir.set(...t,1),t.set(0,0,0)),Ir.applyMatrix4(this.bindMatrix);for(let r=0;r<4;r++){let o=Eu.getComponent(r);if(o!==0){let a=Tu.getComponent(r);Au.multiplyMatrices(n.bones[a].matrixWorld,n.boneInverses[a]),t.addScaledVector(Np.copy(Ir).applyMatrix4(Au),o)}}return t.isVector4&&(t.w=Ir.w),t.applyMatrix4(this.bindMatrixInverse)}},Qs=class extends At{constructor(){super(),this.isBone=!0,this.type="Bone"}},Gi=class extends Tt{constructor(e=null,t=1,n=1,s,r,o,a,c,l=St,h=St,u,f){super(null,o,a,c,l,h,s,r,u,f),this.isDataTexture=!0,this.image={data:e,width:t,height:n},this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}},Ru=new Je,Up=new Je,Yr=class i{constructor(e=[],t=[]){this.uuid=Tn(),this.bones=e.slice(0),this.boneInverses=t,this.boneMatrices=null,this.boneTexture=null,this.init()}init(){let e=this.bones,t=this.boneInverses;if(this.boneMatrices=new Float32Array(e.length*16),t.length===0)this.calculateInverses();else if(e.length!==t.length){ke("Skeleton: Number of inverse bone matrices does not match amount of bones."),this.boneInverses=[];for(let n=0,s=this.bones.length;n<s;n++)this.boneInverses.push(new Je)}}calculateInverses(){this.boneInverses.length=0;for(let e=0,t=this.bones.length;e<t;e++){let n=new Je;this.bones[e]&&n.copy(this.bones[e].matrixWorld).invert(),this.boneInverses.push(n)}}pose(){for(let e=0,t=this.bones.length;e<t;e++){let n=this.bones[e];n&&n.matrixWorld.copy(this.boneInverses[e]).invert()}for(let e=0,t=this.bones.length;e<t;e++){let n=this.bones[e];n&&(n.parent&&n.parent.isBone?(n.matrix.copy(n.parent.matrixWorld).invert(),n.matrix.multiply(n.matrixWorld)):n.matrix.copy(n.matrixWorld),n.matrix.decompose(n.position,n.quaternion,n.scale))}}update(){let e=this.bones,t=this.boneInverses,n=this.boneMatrices,s=this.boneTexture;for(let r=0,o=e.length;r<o;r++){let a=e[r]?e[r].matrixWorld:Up;Ru.multiplyMatrices(a,t[r]),Ru.toArray(n,r*16)}s!==null&&(s.needsUpdate=!0)}clone(){return new i(this.bones,this.boneInverses)}computeBoneTexture(){let e=Math.sqrt(this.bones.length*4);e=Math.ceil(e/4)*4,e=Math.max(e,4);let t=new Float32Array(e*e*4);t.set(this.boneMatrices);let n=new Gi(t,e,e,vn,nn);return n.needsUpdate=!0,this.boneMatrices=t,this.boneTexture=n,this}getBoneByName(e){for(let t=0,n=this.bones.length;t<n;t++){let s=this.bones[t];if(s.name===e)return s}}dispose(){this.boneTexture!==null&&(this.boneTexture.dispose(),this.boneTexture=null)}fromJSON(e,t){this.uuid=e.uuid;for(let n=0,s=e.bones.length;n<s;n++){let r=e.bones[n],o=t[r];o===void 0&&(ke("Skeleton: No bone found with UUID:",r),o=new Qs),this.bones.push(o),this.boneInverses.push(new Je().fromArray(e.boneInverses[n]))}return this.init(),this}toJSON(){let e={metadata:{version:4.7,type:"Skeleton",generator:"Skeleton.toJSON"},bones:[],boneInverses:[]};e.uuid=this.uuid;let t=this.bones,n=this.boneInverses;for(let s=0,r=t.length;s<r;s++){let o=t[s];e.bones.push(o.uuid);let a=n[s];e.boneInverses.push(a.toArray())}return e}},Wi=class extends yt{constructor(e,t,n,s=1){super(e,t,n),this.isInstancedBufferAttribute=!0,this.meshPerAttribute=s}copy(e){return super.copy(e),this.meshPerAttribute=e.meshPerAttribute,this}toJSON(){let e=super.toJSON();return e.meshPerAttribute=this.meshPerAttribute,e.isInstancedBufferAttribute=!0,e}},Gs=new Je,Cu=new Je,Zo=[],Pu=new Xt,Op=new Je,Lr=new ht,Dr=new rn,Zr=class extends ht{constructor(e,t,n){super(e,t),this.isInstancedMesh=!0,this.instanceMatrix=new Wi(new Float32Array(n*16),16),this.instanceColor=null,this.morphTexture=null,this.count=n,this.boundingBox=null,this.boundingSphere=null;for(let s=0;s<n;s++)this.setMatrixAt(s,Op)}computeBoundingBox(){let e=this.geometry,t=this.count;this.boundingBox===null&&(this.boundingBox=new Xt),e.boundingBox===null&&e.computeBoundingBox(),this.boundingBox.makeEmpty();for(let n=0;n<t;n++)this.getMatrixAt(n,Gs),Pu.copy(e.boundingBox).applyMatrix4(Gs),this.boundingBox.union(Pu)}computeBoundingSphere(){let e=this.geometry,t=this.count;this.boundingSphere===null&&(this.boundingSphere=new rn),e.boundingSphere===null&&e.computeBoundingSphere(),this.boundingSphere.makeEmpty();for(let n=0;n<t;n++)this.getMatrixAt(n,Gs),Dr.copy(e.boundingSphere).applyMatrix4(Gs),this.boundingSphere.union(Dr)}copy(e,t){return super.copy(e,t),this.instanceMatrix.copy(e.instanceMatrix),e.morphTexture!==null&&(this.morphTexture=e.morphTexture.clone()),e.instanceColor!==null&&(this.instanceColor=e.instanceColor.clone()),this.count=e.count,e.boundingBox!==null&&(this.boundingBox=e.boundingBox.clone()),e.boundingSphere!==null&&(this.boundingSphere=e.boundingSphere.clone()),this}getColorAt(e,t){return this.instanceColor===null?t.setRGB(1,1,1):t.fromArray(this.instanceColor.array,e*3)}getMatrixAt(e,t){return t.fromArray(this.instanceMatrix.array,e*16)}getMorphAt(e,t){let n=t.morphTargetInfluences,s=this.morphTexture.source.data.data,r=n.length+1,o=e*r+1;for(let a=0;a<n.length;a++)n[a]=s[o+a]}raycast(e,t){let n=this.matrixWorld,s=this.count;if(Lr.geometry=this.geometry,Lr.material=this.material,Lr.material!==void 0&&(this.boundingSphere===null&&this.computeBoundingSphere(),Dr.copy(this.boundingSphere),Dr.applyMatrix4(n),e.ray.intersectsSphere(Dr)!==!1))for(let r=0;r<s;r++){this.getMatrixAt(r,Gs),Cu.multiplyMatrices(n,Gs),Lr.matrixWorld=Cu,Lr.raycast(e,Zo);for(let o=0,a=Zo.length;o<a;o++){let c=Zo[o];c.instanceId=r,c.object=this,t.push(c)}Zo.length=0}}setColorAt(e,t){return this.instanceColor===null&&(this.instanceColor=new Wi(new Float32Array(this.instanceMatrix.count*3).fill(1),3)),t.toArray(this.instanceColor.array,e*3),this}setMatrixAt(e,t){return t.toArray(this.instanceMatrix.array,e*16),this}setMorphAt(e,t){let n=t.morphTargetInfluences,s=n.length+1;this.morphTexture===null&&(this.morphTexture=new Gi(new Float32Array(s*this.count),s,this.count,fr,nn));let r=this.morphTexture.source.data.data,o=0;for(let l=0;l<n.length;l++)o+=n[l];let a=this.geometry.morphTargetsRelative?1:1-o,c=s*e;return r[c]=a,r.set(n,c+1),this}updateMorphTargets(){}dispose(){this.dispatchEvent({type:"dispose"}),this.morphTexture!==null&&(this.morphTexture.dispose(),this.morphTexture=null)}},cl=new B,Fp=new B,Bp=new Ze,Sn=class{constructor(e=new B(1,0,0),t=0){this.isPlane=!0,this.normal=e,this.constant=t}set(e,t){return this.normal.copy(e),this.constant=t,this}setComponents(e,t,n,s){return this.normal.set(e,t,n),this.constant=s,this}setFromNormalAndCoplanarPoint(e,t){return this.normal.copy(e),this.constant=-t.dot(this.normal),this}setFromCoplanarPoints(e,t,n){let s=cl.subVectors(n,t).cross(Fp.subVectors(e,t)).normalize();return this.setFromNormalAndCoplanarPoint(s,e),this}copy(e){return this.normal.copy(e.normal),this.constant=e.constant,this}normalize(){let e=1/this.normal.length();return this.normal.multiplyScalar(e),this.constant*=e,this}negate(){return this.constant*=-1,this.normal.negate(),this}distanceToPoint(e){return this.normal.dot(e)+this.constant}distanceToSphere(e){return this.distanceToPoint(e.center)-e.radius}projectPoint(e,t){return t.copy(e).addScaledVector(this.normal,-this.distanceToPoint(e))}intersectLine(e,t,n=!0){let s=e.delta(cl),r=this.normal.dot(s);if(r===0)return this.distanceToPoint(e.start)===0?t.copy(e.start):null;let o=-(e.start.dot(this.normal)+this.constant)/r;return n===!0&&(o<0||o>1)?null:t.copy(e.start).addScaledVector(s,o)}intersectsLine(e){let t=this.distanceToPoint(e.start),n=this.distanceToPoint(e.end);return t<0&&n>0||n<0&&t>0}intersectsBox(e){return e.intersectsPlane(this)}intersectsSphere(e){return e.intersectsPlane(this)}coplanarPoint(e){return e.copy(this.normal).multiplyScalar(-this.constant)}applyMatrix4(e,t){let n=t||Bp.getNormalMatrix(e),s=this.coplanarPoint(cl).applyMatrix4(e),r=this.normal.applyMatrix3(n).normalize();return this.constant=-s.dot(r),this}translate(e){return this.constant-=e.dot(this.normal),this}equals(e){return e.normal.equals(this.normal)&&e.constant===this.constant}clone(){return new this.constructor().copy(this)}},as=new rn,kp=new de(.5,.5),Ko=new B,er=class{constructor(e=new Sn,t=new Sn,n=new Sn,s=new Sn,r=new Sn,o=new Sn){this.planes=[e,t,n,s,r,o]}set(e,t,n,s,r,o){let a=this.planes;return a[0].copy(e),a[1].copy(t),a[2].copy(n),a[3].copy(s),a[4].copy(r),a[5].copy(o),this}copy(e){let t=this.planes;for(let n=0;n<6;n++)t[n].copy(e.planes[n]);return this}setFromProjectionMatrix(e,t=Un,n=!1){let s=this.planes,r=e.elements,o=r[0],a=r[1],c=r[2],l=r[3],h=r[4],u=r[5],f=r[6],d=r[7],p=r[8],y=r[9],m=r[10],g=r[11],E=r[12],S=r[13],x=r[14],A=r[15];if(s[0].setComponents(l-o,d-h,g-p,A-E).normalize(),s[1].setComponents(l+o,d+h,g+p,A+E).normalize(),s[2].setComponents(l+a,d+u,g+y,A+S).normalize(),s[3].setComponents(l-a,d-u,g-y,A-S).normalize(),n)s[4].setComponents(c,f,m,x).normalize(),s[5].setComponents(l-c,d-f,g-m,A-x).normalize();else if(s[4].setComponents(l-c,d-f,g-m,A-x).normalize(),t===Un)s[5].setComponents(l+c,d+f,g+m,A+x).normalize();else if(t===Zs)s[5].setComponents(c,f,m,x).normalize();else throw new Error("THREE.Frustum.setFromProjectionMatrix(): Invalid coordinate system: "+t);return this}intersectsObject(e){if(e.boundingSphere!==void 0)e.boundingSphere===null&&e.computeBoundingSphere(),as.copy(e.boundingSphere).applyMatrix4(e.matrixWorld);else{let t=e.geometry;t.boundingSphere===null&&t.computeBoundingSphere(),as.copy(t.boundingSphere).applyMatrix4(e.matrixWorld)}return this.intersectsSphere(as)}intersectsSprite(e){as.center.set(0,0,0);let t=kp.distanceTo(e.center);return as.radius=.7071067811865476+t,as.applyMatrix4(e.matrixWorld),this.intersectsSphere(as)}intersectsSphere(e){let t=this.planes,n=e.center,s=-e.radius;for(let r=0;r<6;r++)if(t[r].distanceToPoint(n)<s)return!1;return!0}intersectsBox(e){let t=this.planes;for(let n=0;n<6;n++){let s=t[n];if(Ko.x=s.normal.x>0?e.max.x:e.min.x,Ko.y=s.normal.y>0?e.max.y:e.min.y,Ko.z=s.normal.z>0?e.max.z:e.min.z,s.distanceToPoint(Ko)<0)return!1}return!0}containsPoint(e){let t=this.planes;for(let n=0;n<6;n++)if(t[n].distanceToPoint(e)<0)return!1;return!0}clone(){return new this.constructor().copy(this)}};var tr=class extends on{constructor(e){super(),this.isLineBasicMaterial=!0,this.type="LineBasicMaterial",this.color=new Ge(16777215),this.map=null,this.linewidth=1,this.linecap="round",this.linejoin="round",this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.linewidth=e.linewidth,this.linecap=e.linecap,this.linejoin=e.linejoin,this.fog=e.fog,this}},ya=new B,va=new B,Iu=new Je,Nr=new _i,jo=new rn,ll=new B,Lu=new B,ms=class extends At{constructor(e=new wt,t=new tr){super(),this.isLine=!0,this.type="Line",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}computeLineDistances(){let e=this.geometry;if(e.index===null){let t=e.attributes.position,n=[0];for(let s=1,r=t.count;s<r;s++)ya.fromBufferAttribute(t,s-1),va.fromBufferAttribute(t,s),n[s]=n[s-1],n[s]+=ya.distanceTo(va);e.setAttribute("lineDistance",new Mt(n,1))}else ke("Line.computeLineDistances(): Computation only possible with non-indexed BufferGeometry.");return this}raycast(e,t){let n=this.geometry,s=this.matrixWorld,r=e.params.Line.threshold,o=n.drawRange;if(n.boundingSphere===null&&n.computeBoundingSphere(),jo.copy(n.boundingSphere),jo.applyMatrix4(s),jo.radius+=r,e.ray.intersectsSphere(jo)===!1)return;Iu.copy(s).invert(),Nr.copy(e.ray).applyMatrix4(Iu);let a=r/((this.scale.x+this.scale.y+this.scale.z)/3),c=a*a,l=this.isLineSegments?2:1,h=n.index,f=n.attributes.position;if(h!==null){let d=Math.max(0,o.start),p=Math.min(h.count,o.start+o.count);for(let y=d,m=p-1;y<m;y+=l){let g=h.getX(y),E=h.getX(y+1),S=Jo(this,e,Nr,c,g,E,y);S&&t.push(S)}if(this.isLineLoop){let y=h.getX(p-1),m=h.getX(d),g=Jo(this,e,Nr,c,y,m,p-1);g&&t.push(g)}}else{let d=Math.max(0,o.start),p=Math.min(f.count,o.start+o.count);for(let y=d,m=p-1;y<m;y+=l){let g=Jo(this,e,Nr,c,y,y+1,y);g&&t.push(g)}if(this.isLineLoop){let y=Jo(this,e,Nr,c,p-1,d,p-1);y&&t.push(y)}}}updateMorphTargets(){let t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){let s=t[n[0]];if(s!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let r=0,o=s.length;r<o;r++){let a=s[r].name||String(r);this.morphTargetInfluences.push(0),this.morphTargetDictionary[a]=r}}}}};function Jo(i,e,t,n,s,r,o){let a=i.geometry.attributes.position;if(ya.fromBufferAttribute(a,s),va.fromBufferAttribute(a,r),t.distanceSqToSegment(ya,va,ll,Lu)>n)return;ll.applyMatrix4(i.matrixWorld);let l=e.ray.origin.distanceTo(ll);if(!(l<e.near||l>e.far))return{distance:l,point:Lu.clone().applyMatrix4(i.matrixWorld),index:o,face:null,faceIndex:null,barycoord:null,object:i}}var Du=new B,Nu=new B,Kr=class extends ms{constructor(e,t){super(e,t),this.isLineSegments=!0,this.type="LineSegments"}computeLineDistances(){let e=this.geometry;if(e.index===null){let t=e.attributes.position,n=[];for(let s=0,r=t.count;s<r;s+=2)Du.fromBufferAttribute(t,s),Nu.fromBufferAttribute(t,s+1),n[s]=s===0?0:n[s-1],n[s+1]=n[s]+Du.distanceTo(Nu);e.setAttribute("lineDistance",new Mt(n,1))}else ke("LineSegments.computeLineDistances(): Computation only possible with non-indexed BufferGeometry.");return this}},jr=class extends ms{constructor(e,t){super(e,t),this.isLineLoop=!0,this.type="LineLoop"}},nr=class extends on{constructor(e){super(),this.isPointsMaterial=!0,this.type="PointsMaterial",this.color=new Ge(16777215),this.map=null,this.alphaMap=null,this.size=1,this.sizeAttenuation=!0,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.alphaMap=e.alphaMap,this.size=e.size,this.sizeAttenuation=e.sizeAttenuation,this.fog=e.fog,this}},Uu=new Je,bl=new _i,$o=new rn,Qo=new B,Jr=class extends At{constructor(e=new wt,t=new nr){super(),this.isPoints=!0,this.type="Points",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}raycast(e,t){let n=this.geometry,s=this.matrixWorld,r=e.params.Points.threshold,o=n.drawRange;if(n.boundingSphere===null&&n.computeBoundingSphere(),$o.copy(n.boundingSphere),$o.applyMatrix4(s),$o.radius+=r,e.ray.intersectsSphere($o)===!1)return;Uu.copy(s).invert(),bl.copy(e.ray).applyMatrix4(Uu);let a=r/((this.scale.x+this.scale.y+this.scale.z)/3),c=a*a,l=n.index,u=n.attributes.position;if(l!==null){let f=Math.max(0,o.start),d=Math.min(l.count,o.start+o.count);for(let p=f,y=d;p<y;p++){let m=l.getX(p);Qo.fromBufferAttribute(u,m),Ou(Qo,m,c,s,e,t,this)}}else{let f=Math.max(0,o.start),d=Math.min(u.count,o.start+o.count);for(let p=f,y=d;p<y;p++)Qo.fromBufferAttribute(u,p),Ou(Qo,p,c,s,e,t,this)}}updateMorphTargets(){let t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){let s=t[n[0]];if(s!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let r=0,o=s.length;r<o;r++){let a=s[r].name||String(r);this.morphTargetInfluences.push(0),this.morphTargetDictionary[a]=r}}}}};function Ou(i,e,t,n,s,r,o){let a=bl.distanceSqToPoint(i);if(a<t){let c=new B;bl.closestPointToPoint(i,c),c.applyMatrix4(n);let l=s.ray.origin.distanceTo(c);if(l<s.near||l>s.far)return;r.push({distance:l,distanceToRay:Math.sqrt(a),point:c,index:e,face:null,faceIndex:null,barycoord:null,object:o})}}var $r=class extends Tt{constructor(e=[],t=Ki,n,s,r,o,a,c,l,h){super(e,t,n,s,r,o,a,c,l,h),this.isCubeTexture=!0,this.flipY=!1}get images(){return this.image}set images(e){this.image=e}},Qn=class extends Tt{constructor(e,t,n,s,r,o,a,c,l){super(e,t,n,s,r,o,a,c,l),this.isCanvasTexture=!0,this.needsUpdate=!0}};var xi=class extends Tt{constructor(e,t,n=Wn,s,r,o,a=St,c=St,l,h=$n,u=1){if(h!==$n&&h!==ji)throw new Error("THREE.DepthTexture: format must be either THREE.DepthFormat or THREE.DepthStencilFormat");let f={width:e,height:t,depth:u};super(f,s,r,o,a,c,h,n,l),this.isDepthTexture=!0,this.flipY=!1,this.generateMipmaps=!1,this.compareFunction=null}copy(e){return super.copy(e),this.source=new Js(Object.assign({},e.image)),this.compareFunction=e.compareFunction,this}toJSON(e){let t=super.toJSON(e);return this.compareFunction!==null&&(t.compareFunction=this.compareFunction),t}},ba=class extends xi{constructor(e,t=Wn,n=Ki,s,r,o=St,a=St,c,l=$n){let h={width:e,height:e,depth:1},u=[h,h,h,h,h,h];super(e,e,t,n,s,r,o,a,c,l),this.image=u,this.isCubeDepthTexture=!0,this.isCubeTexture=!0}get images(){return this.image}set images(e){this.image=e}},Qr=class extends Tt{constructor(e=null){super(),this.sourceTexture=e,this.isExternalTexture=!0}copy(e){return super.copy(e),this.sourceTexture=e.sourceTexture,this}},En=class i extends wt{constructor(e=1,t=1,n=1,s=1,r=1,o=1){super(),this.type="BoxGeometry",this.parameters={width:e,height:t,depth:n,widthSegments:s,heightSegments:r,depthSegments:o};let a=this;s=Math.floor(s),r=Math.floor(r),o=Math.floor(o);let c=[],l=[],h=[],u=[],f=0,d=0;p("z","y","x",-1,-1,n,t,e,o,r,0),p("z","y","x",1,-1,n,t,-e,o,r,1),p("x","z","y",1,1,e,n,t,s,o,2),p("x","z","y",1,-1,e,n,-t,s,o,3),p("x","y","z",1,-1,e,t,n,s,r,4),p("x","y","z",-1,-1,e,t,-n,s,r,5),this.setIndex(c),this.setAttribute("position",new Mt(l,3)),this.setAttribute("normal",new Mt(h,3)),this.setAttribute("uv",new Mt(u,2));function p(y,m,g,E,S,x,A,w,I,v,R){let F=x/I,N=A/v,H=x/2,oe=A/2,ie=w/2,X=I+1,ee=v+1,Y=0,G=0,ge=new B;for(let ye=0;ye<ee;ye++){let Me=ye*N-oe;for(let Re=0;Re<X;Re++){let ze=Re*F-H;ge[y]=ze*E,ge[m]=Me*S,ge[g]=ie,l.push(ge.x,ge.y,ge.z),ge[y]=0,ge[m]=0,ge[g]=w>0?1:-1,h.push(ge.x,ge.y,ge.z),u.push(Re/I),u.push(1-ye/v),Y+=1}}for(let ye=0;ye<v;ye++)for(let Me=0;Me<I;Me++){let Re=f+Me+X*ye,ze=f+Me+X*(ye+1),qe=f+(Me+1)+X*(ye+1),We=f+(Me+1)+X*ye;c.push(Re,ze,We),c.push(ze,qe,We),G+=6}a.addGroup(d,G,R),d+=G,f+=Y}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new i(e.width,e.height,e.depth,e.widthSegments,e.heightSegments,e.depthSegments)}};var gn=class{constructor(){this.type="Curve",this.arcLengthDivisions=200,this.needsUpdate=!1,this.cacheArcLengths=null}getPoint(){ke("Curve: .getPoint() not implemented.")}getPointAt(e,t){let n=this.getUtoTmapping(e);return this.getPoint(n,t)}getPoints(e=5){let t=[];for(let n=0;n<=e;n++)t.push(this.getPoint(n/e));return t}getSpacedPoints(e=5){let t=[];for(let n=0;n<=e;n++)t.push(this.getPointAt(n/e));return t}getLength(){let e=this.getLengths();return e[e.length-1]}getLengths(e=this.arcLengthDivisions){if(this.cacheArcLengths&&this.cacheArcLengths.length===e+1&&!this.needsUpdate)return this.cacheArcLengths;this.needsUpdate=!1;let t=[],n,s=this.getPoint(0),r=0;t.push(0);for(let o=1;o<=e;o++)n=this.getPoint(o/e),r+=n.distanceTo(s),t.push(r),s=n;return this.cacheArcLengths=t,t}updateArcLengths(){this.needsUpdate=!0,this.getLengths()}getUtoTmapping(e,t=null){let n=this.getLengths(),s=0,r=n.length,o;t?o=t:o=e*n[r-1];let a=0,c=r-1,l;for(;a<=c;)if(s=Math.floor(a+(c-a)/2),l=n[s]-o,l<0)a=s+1;else if(l>0)c=s-1;else{c=s;break}if(s=c,n[s]===o)return s/(r-1);let h=n[s],f=n[s+1]-h,d=(o-h)/f;return(s+d)/(r-1)}getTangent(e,t){let s=e-1e-4,r=e+1e-4;s<0&&(s=0),r>1&&(r=1);let o=this.getPoint(s),a=this.getPoint(r),c=t||(o.isVector2?new de:new B);return c.copy(a).sub(o).normalize(),c}getTangentAt(e,t){let n=this.getUtoTmapping(e);return this.getTangent(n,t)}computeFrenetFrames(e,t=!1){let n=new B,s=[],r=[],o=[],a=new B,c=new Je;for(let d=0;d<=e;d++){let p=d/e;s[d]=this.getTangentAt(p,new B)}r[0]=new B,o[0]=new B;let l=Number.MAX_VALUE,h=Math.abs(s[0].x),u=Math.abs(s[0].y),f=Math.abs(s[0].z);h<=l&&(l=h,n.set(1,0,0)),u<=l&&(l=u,n.set(0,1,0)),f<=l&&n.set(0,0,1),a.crossVectors(s[0],n).normalize(),r[0].crossVectors(s[0],a),o[0].crossVectors(s[0],r[0]);for(let d=1;d<=e;d++){if(r[d]=r[d-1].clone(),o[d]=o[d-1].clone(),a.crossVectors(s[d-1],s[d]),a.length()>Number.EPSILON){a.normalize();let p=Math.acos(et(s[d-1].dot(s[d]),-1,1));r[d].applyMatrix4(c.makeRotationAxis(a,p))}o[d].crossVectors(s[d],r[d])}if(t===!0){let d=Math.acos(et(r[0].dot(r[e]),-1,1));d/=e,s[0].dot(a.crossVectors(r[0],r[e]))>0&&(d=-d);for(let p=1;p<=e;p++)r[p].applyMatrix4(c.makeRotationAxis(s[p],d*p)),o[p].crossVectors(s[p],r[p])}return{tangents:s,normals:r,binormals:o}}clone(){return new this.constructor().copy(this)}copy(e){return this.arcLengthDivisions=e.arcLengthDivisions,this}toJSON(){let e={metadata:{version:4.7,type:"Curve",generator:"Curve.toJSON"}};return e.arcLengthDivisions=this.arcLengthDivisions,e.type=this.type,e}fromJSON(e){return this.arcLengthDivisions=e.arcLengthDivisions,this}},ir=class extends gn{constructor(e=0,t=0,n=1,s=1,r=0,o=Math.PI*2,a=!1,c=0){super(),this.isEllipseCurve=!0,this.type="EllipseCurve",this.aX=e,this.aY=t,this.xRadius=n,this.yRadius=s,this.aStartAngle=r,this.aEndAngle=o,this.aClockwise=a,this.aRotation=c}getPoint(e,t=new de){let n=t,s=Math.PI*2,r=this.aEndAngle-this.aStartAngle,o=Math.abs(r)<Number.EPSILON;for(;r<0;)r+=s;for(;r>s;)r-=s;r<Number.EPSILON&&(o?r=0:r=s),this.aClockwise===!0&&!o&&(r===s?r=-s:r=r-s);let a=this.aStartAngle+e*r,c=this.aX+this.xRadius*Math.cos(a),l=this.aY+this.yRadius*Math.sin(a);if(this.aRotation!==0){let h=Math.cos(this.aRotation),u=Math.sin(this.aRotation),f=c-this.aX,d=l-this.aY;c=f*h-d*u+this.aX,l=f*u+d*h+this.aY}return n.set(c,l)}copy(e){return super.copy(e),this.aX=e.aX,this.aY=e.aY,this.xRadius=e.xRadius,this.yRadius=e.yRadius,this.aStartAngle=e.aStartAngle,this.aEndAngle=e.aEndAngle,this.aClockwise=e.aClockwise,this.aRotation=e.aRotation,this}toJSON(){let e=super.toJSON();return e.aX=this.aX,e.aY=this.aY,e.xRadius=this.xRadius,e.yRadius=this.yRadius,e.aStartAngle=this.aStartAngle,e.aEndAngle=this.aEndAngle,e.aClockwise=this.aClockwise,e.aRotation=this.aRotation,e}fromJSON(e){return super.fromJSON(e),this.aX=e.aX,this.aY=e.aY,this.xRadius=e.xRadius,this.yRadius=e.yRadius,this.aStartAngle=e.aStartAngle,this.aEndAngle=e.aEndAngle,this.aClockwise=e.aClockwise,this.aRotation=e.aRotation,this}},Ma=class extends ir{constructor(e,t,n,s,r,o){super(e,t,n,n,s,r,o),this.isArcCurve=!0,this.type="ArcCurve"}};function $l(){let i=0,e=0,t=0,n=0;function s(r,o,a,c){i=r,e=a,t=-3*r+3*o-2*a-c,n=2*r-2*o+a+c}return{initCatmullRom:function(r,o,a,c,l){s(o,a,l*(a-r),l*(c-o))},initNonuniformCatmullRom:function(r,o,a,c,l,h,u){let f=(o-r)/l-(a-r)/(l+h)+(a-o)/h,d=(a-o)/h-(c-o)/(h+u)+(c-a)/u;f*=h,d*=h,s(o,a,f,d)},calc:function(r){let o=r*r,a=o*r;return i+e*r+t*o+n*a}}}var Fu=new B,Bu=new B,hl=new $l,ul=new $l,fl=new $l,Sa=class extends gn{constructor(e=[],t=!1,n="centripetal",s=.5){super(),this.isCatmullRomCurve3=!0,this.type="CatmullRomCurve3",this.points=e,this.closed=t,this.curveType=n,this.tension=s}getPoint(e,t=new B){let n=t,s=this.points,r=s.length,o=(r-(this.closed?0:1))*e,a=Math.floor(o),c=o-a;this.closed?a+=a>0?0:(Math.floor(Math.abs(a)/r)+1)*r:c===0&&a===r-1&&(a=r-2,c=1);let l,h;this.closed||a>0?l=s[(a-1)%r]:(Bu.subVectors(s[0],s[1]).add(s[0]),l=Bu);let u=s[a%r],f=s[(a+1)%r];if(this.closed||a+2<r?h=s[(a+2)%r]:(Fu.subVectors(s[r-1],s[r-2]).add(s[r-1]),h=Fu),this.curveType==="centripetal"||this.curveType==="chordal"){let d=this.curveType==="chordal"?.5:.25,p=Math.pow(l.distanceToSquared(u),d),y=Math.pow(u.distanceToSquared(f),d),m=Math.pow(f.distanceToSquared(h),d);y<1e-4&&(y=1),p<1e-4&&(p=y),m<1e-4&&(m=y),hl.initNonuniformCatmullRom(l.x,u.x,f.x,h.x,p,y,m),ul.initNonuniformCatmullRom(l.y,u.y,f.y,h.y,p,y,m),fl.initNonuniformCatmullRom(l.z,u.z,f.z,h.z,p,y,m)}else this.curveType==="catmullrom"&&(hl.initCatmullRom(l.x,u.x,f.x,h.x,this.tension),ul.initCatmullRom(l.y,u.y,f.y,h.y,this.tension),fl.initCatmullRom(l.z,u.z,f.z,h.z,this.tension));return n.set(hl.calc(c),ul.calc(c),fl.calc(c)),n}copy(e){super.copy(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(s.clone())}return this.closed=e.closed,this.curveType=e.curveType,this.tension=e.tension,this}toJSON(){let e=super.toJSON();e.points=[];for(let t=0,n=this.points.length;t<n;t++){let s=this.points[t];e.points.push(s.toArray())}return e.closed=this.closed,e.curveType=this.curveType,e.tension=this.tension,e}fromJSON(e){super.fromJSON(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(new B().fromArray(s))}return this.closed=e.closed,this.curveType=e.curveType,this.tension=e.tension,this}};function ku(i,e,t,n,s){let r=(n-e)*.5,o=(s-t)*.5,a=i*i,c=i*a;return(2*t-2*n+r+o)*c+(-3*t+3*n-2*r-o)*a+r*i+t}function zp(i,e){let t=1-i;return t*t*e}function Hp(i,e){return 2*(1-i)*i*e}function Vp(i,e){return i*i*e}function Br(i,e,t,n){return zp(i,e)+Hp(i,t)+Vp(i,n)}function Gp(i,e){let t=1-i;return t*t*t*e}function Wp(i,e){let t=1-i;return 3*t*t*i*e}function Xp(i,e){return 3*(1-i)*i*i*e}function qp(i,e){return i*i*i*e}function kr(i,e,t,n,s){return Gp(i,e)+Wp(i,t)+Xp(i,n)+qp(i,s)}var eo=class extends gn{constructor(e=new de,t=new de,n=new de,s=new de){super(),this.isCubicBezierCurve=!0,this.type="CubicBezierCurve",this.v0=e,this.v1=t,this.v2=n,this.v3=s}getPoint(e,t=new de){let n=t,s=this.v0,r=this.v1,o=this.v2,a=this.v3;return n.set(kr(e,s.x,r.x,o.x,a.x),kr(e,s.y,r.y,o.y,a.y)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this.v3.copy(e.v3),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e.v3=this.v3.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this.v3.fromArray(e.v3),this}},Ta=class extends gn{constructor(e=new B,t=new B,n=new B,s=new B){super(),this.isCubicBezierCurve3=!0,this.type="CubicBezierCurve3",this.v0=e,this.v1=t,this.v2=n,this.v3=s}getPoint(e,t=new B){let n=t,s=this.v0,r=this.v1,o=this.v2,a=this.v3;return n.set(kr(e,s.x,r.x,o.x,a.x),kr(e,s.y,r.y,o.y,a.y),kr(e,s.z,r.z,o.z,a.z)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this.v3.copy(e.v3),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e.v3=this.v3.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this.v3.fromArray(e.v3),this}},to=class extends gn{constructor(e=new de,t=new de){super(),this.isLineCurve=!0,this.type="LineCurve",this.v1=e,this.v2=t}getPoint(e,t=new de){let n=t;return e===1?n.copy(this.v2):(n.copy(this.v2).sub(this.v1),n.multiplyScalar(e).add(this.v1)),n}getPointAt(e,t){return this.getPoint(e,t)}getTangent(e,t=new de){return t.subVectors(this.v2,this.v1).normalize()}getTangentAt(e,t){return this.getTangent(e,t)}copy(e){return super.copy(e),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},Ea=class extends gn{constructor(e=new B,t=new B){super(),this.isLineCurve3=!0,this.type="LineCurve3",this.v1=e,this.v2=t}getPoint(e,t=new B){let n=t;return e===1?n.copy(this.v2):(n.copy(this.v2).sub(this.v1),n.multiplyScalar(e).add(this.v1)),n}getPointAt(e,t){return this.getPoint(e,t)}getTangent(e,t=new B){return t.subVectors(this.v2,this.v1).normalize()}getTangentAt(e,t){return this.getTangent(e,t)}copy(e){return super.copy(e),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},no=class extends gn{constructor(e=new de,t=new de,n=new de){super(),this.isQuadraticBezierCurve=!0,this.type="QuadraticBezierCurve",this.v0=e,this.v1=t,this.v2=n}getPoint(e,t=new de){let n=t,s=this.v0,r=this.v1,o=this.v2;return n.set(Br(e,s.x,r.x,o.x),Br(e,s.y,r.y,o.y)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},Aa=class extends gn{constructor(e=new B,t=new B,n=new B){super(),this.isQuadraticBezierCurve3=!0,this.type="QuadraticBezierCurve3",this.v0=e,this.v1=t,this.v2=n}getPoint(e,t=new B){let n=t,s=this.v0,r=this.v1,o=this.v2;return n.set(Br(e,s.x,r.x,o.x),Br(e,s.y,r.y,o.y),Br(e,s.z,r.z,o.z)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},io=class extends gn{constructor(e=[]){super(),this.isSplineCurve=!0,this.type="SplineCurve",this.points=e}getPoint(e,t=new de){let n=t,s=this.points,r=(s.length-1)*e,o=Math.floor(r),a=r-o,c=s[o===0?o:o-1],l=s[o],h=s[o>s.length-2?s.length-1:o+1],u=s[o>s.length-3?s.length-1:o+2];return n.set(ku(a,c.x,l.x,h.x,u.x),ku(a,c.y,l.y,h.y,u.y)),n}copy(e){super.copy(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(s.clone())}return this}toJSON(){let e=super.toJSON();e.points=[];for(let t=0,n=this.points.length;t<n;t++){let s=this.points[t];e.points.push(s.toArray())}return e}fromJSON(e){super.fromJSON(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(new de().fromArray(s))}return this}},Ml=Object.freeze({__proto__:null,ArcCurve:Ma,CatmullRomCurve3:Sa,CubicBezierCurve:eo,CubicBezierCurve3:Ta,EllipseCurve:ir,LineCurve:to,LineCurve3:Ea,QuadraticBezierCurve:no,QuadraticBezierCurve3:Aa,SplineCurve:io}),wa=class extends gn{constructor(){super(),this.type="CurvePath",this.curves=[],this.autoClose=!1}add(e){this.curves.push(e)}closePath(){let e=this.curves[0].getPoint(0),t=this.curves[this.curves.length-1].getPoint(1);if(!e.equals(t)){let n=e.isVector2===!0?"LineCurve":"LineCurve3";this.curves.push(new Ml[n](t,e))}return this}getPoint(e,t){let n=e*this.getLength(),s=this.getCurveLengths(),r=0;for(;r<s.length;){if(s[r]>=n){let o=s[r]-n,a=this.curves[r],c=a.getLength(),l=c===0?0:1-o/c;return a.getPointAt(l,t)}r++}return null}getLength(){let e=this.getCurveLengths();return e[e.length-1]}updateArcLengths(){this.needsUpdate=!0,this.cacheLengths=null,this.getCurveLengths()}getCurveLengths(){if(this.cacheLengths&&this.cacheLengths.length===this.curves.length)return this.cacheLengths;let e=[],t=0;for(let n=0,s=this.curves.length;n<s;n++)t+=this.curves[n].getLength(),e.push(t);return this.cacheLengths=e,e}getSpacedPoints(e=40){let t=[];for(let n=0;n<=e;n++)t.push(this.getPoint(n/e));return this.autoClose&&t.push(t[0]),t}getPoints(e=12){let t=[],n;for(let s=0,r=this.curves;s<r.length;s++){let o=r[s],a=o.isEllipseCurve?e*2:o.isLineCurve||o.isLineCurve3?1:o.isSplineCurve?e*o.points.length:e,c=o.getPoints(a);for(let l=0;l<c.length;l++){let h=c[l];n&&n.equals(h)||(t.push(h),n=h)}}return this.autoClose&&t.length>1&&!t[t.length-1].equals(t[0])&&t.push(t[0]),t}copy(e){super.copy(e),this.curves=[];for(let t=0,n=e.curves.length;t<n;t++){let s=e.curves[t];this.curves.push(s.clone())}return this.autoClose=e.autoClose,this}toJSON(){let e=super.toJSON();e.autoClose=this.autoClose,e.curves=[];for(let t=0,n=this.curves.length;t<n;t++){let s=this.curves[t];e.curves.push(s.toJSON())}return e}fromJSON(e){super.fromJSON(e),this.autoClose=e.autoClose,this.curves=[];for(let t=0,n=e.curves.length;t<n;t++){let s=e.curves[t];this.curves.push(new Ml[s.type]().fromJSON(s))}return this}},tn=class extends wa{constructor(e){super(),this.type="Path",this.currentPoint=new de,e&&this.setFromPoints(e)}setFromPoints(e){this.moveTo(e[0].x,e[0].y);for(let t=1,n=e.length;t<n;t++)this.lineTo(e[t].x,e[t].y);return this}moveTo(e,t){return this.currentPoint.set(e,t),this}lineTo(e,t){let n=new to(this.currentPoint.clone(),new de(e,t));return this.curves.push(n),this.currentPoint.set(e,t),this}quadraticCurveTo(e,t,n,s){let r=new no(this.currentPoint.clone(),new de(e,t),new de(n,s));return this.curves.push(r),this.currentPoint.set(n,s),this}bezierCurveTo(e,t,n,s,r,o){let a=new eo(this.currentPoint.clone(),new de(e,t),new de(n,s),new de(r,o));return this.curves.push(a),this.currentPoint.set(r,o),this}splineThru(e){let t=[this.currentPoint.clone()].concat(e),n=new io(t);return this.curves.push(n),this.currentPoint.copy(e[e.length-1]),this}arc(e,t,n,s,r,o){let a=this.currentPoint.x,c=this.currentPoint.y;return this.absarc(e+a,t+c,n,s,r,o),this}absarc(e,t,n,s,r,o){return this.absellipse(e,t,n,n,s,r,o),this}ellipse(e,t,n,s,r,o,a,c){let l=this.currentPoint.x,h=this.currentPoint.y;return this.absellipse(e+l,t+h,n,s,r,o,a,c),this}absellipse(e,t,n,s,r,o,a,c){let l=new ir(e,t,n,s,r,o,a,c);if(this.curves.length>0){let u=l.getPoint(0);u.equals(this.currentPoint)||this.lineTo(u.x,u.y)}this.curves.push(l);let h=l.getPoint(1);return this.currentPoint.copy(h),this}copy(e){return super.copy(e),this.currentPoint.copy(e.currentPoint),this}toJSON(){let e=super.toJSON();return e.currentPoint=this.currentPoint.toArray(),e}fromJSON(e){return super.fromJSON(e),this.currentPoint.fromArray(e.currentPoint),this}},kn=class extends tn{constructor(e){super(e),this.uuid=Tn(),this.type="Shape",this.holes=[]}getPointsHoles(e){let t=[];for(let n=0,s=this.holes.length;n<s;n++)t[n]=this.holes[n].getPoints(e);return t}extractPoints(e){return{shape:this.getPoints(e),holes:this.getPointsHoles(e)}}copy(e){super.copy(e),this.holes=[];for(let t=0,n=e.holes.length;t<n;t++){let s=e.holes[t];this.holes.push(s.clone())}return this}toJSON(){let e=super.toJSON();e.uuid=this.uuid,e.holes=[];for(let t=0,n=this.holes.length;t<n;t++){let s=this.holes[t];e.holes.push(s.toJSON())}return e}fromJSON(e){super.fromJSON(e),this.uuid=e.uuid,this.holes=[];for(let t=0,n=e.holes.length;t<n;t++){let s=e.holes[t];this.holes.push(new tn().fromJSON(s))}return this}};function Yp(i,e,t=2){let n=e&&e.length,s=n?e[0]*t:i.length,r=Of(i,0,s,t,!0),o=[];if(!r||r.next===r.prev)return o;let a,c,l;if(n&&(r=$p(i,e,r,t)),i.length>80*t){a=i[0],c=i[1];let h=a,u=c;for(let f=t;f<s;f+=t){let d=i[f],p=i[f+1];d<a&&(a=d),p<c&&(c=p),d>h&&(h=d),p>u&&(u=p)}l=Math.max(h-a,u-c),l=l!==0?32767/l:0}return so(r,o,t,a,c,l,0),o}function Of(i,e,t,n,s){let r;if(s===lm(i,e,t,n)>0)for(let o=e;o<t;o+=n)r=zu(o/n|0,i[o],i[o+1],r);else for(let o=t-n;o>=e;o-=n)r=zu(o/n|0,i[o],i[o+1],r);return r&&sr(r,r.next)&&(oo(r),r=r.next),r}function gs(i,e){if(!i)return i;e||(e=i);let t=i,n;do if(n=!1,!t.steiner&&(sr(t,t.next)||Et(t.prev,t,t.next)===0)){if(oo(t),t=e=t.prev,t===t.next)break;n=!0}else t=t.next;while(n||t!==e);return e}function so(i,e,t,n,s,r,o){if(!i)return;!o&&r&&im(i,n,s,r);let a=i;for(;i.prev!==i.next;){let c=i.prev,l=i.next;if(r?Kp(i,n,s,r):Zp(i)){e.push(c.i,i.i,l.i),oo(i),i=l.next,a=l.next;continue}if(i=l,i===a){o?o===1?(i=jp(gs(i),e),so(i,e,t,n,s,r,2)):o===2&&Jp(i,e,t,n,s,r):so(gs(i),e,t,n,s,r,1);break}}}function Zp(i){let e=i.prev,t=i,n=i.next;if(Et(e,t,n)>=0)return!1;let s=e.x,r=t.x,o=n.x,a=e.y,c=t.y,l=n.y,h=Math.min(s,r,o),u=Math.min(a,c,l),f=Math.max(s,r,o),d=Math.max(a,c,l),p=n.next;for(;p!==e;){if(p.x>=h&&p.x<=f&&p.y>=u&&p.y<=d&&Ur(s,a,r,c,o,l,p.x,p.y)&&Et(p.prev,p,p.next)>=0)return!1;p=p.next}return!0}function Kp(i,e,t,n){let s=i.prev,r=i,o=i.next;if(Et(s,r,o)>=0)return!1;let a=s.x,c=r.x,l=o.x,h=s.y,u=r.y,f=o.y,d=Math.min(a,c,l),p=Math.min(h,u,f),y=Math.max(a,c,l),m=Math.max(h,u,f),g=Sl(d,p,e,t,n),E=Sl(y,m,e,t,n),S=i.prevZ,x=i.nextZ;for(;S&&S.z>=g&&x&&x.z<=E;){if(S.x>=d&&S.x<=y&&S.y>=p&&S.y<=m&&S!==s&&S!==o&&Ur(a,h,c,u,l,f,S.x,S.y)&&Et(S.prev,S,S.next)>=0||(S=S.prevZ,x.x>=d&&x.x<=y&&x.y>=p&&x.y<=m&&x!==s&&x!==o&&Ur(a,h,c,u,l,f,x.x,x.y)&&Et(x.prev,x,x.next)>=0))return!1;x=x.nextZ}for(;S&&S.z>=g;){if(S.x>=d&&S.x<=y&&S.y>=p&&S.y<=m&&S!==s&&S!==o&&Ur(a,h,c,u,l,f,S.x,S.y)&&Et(S.prev,S,S.next)>=0)return!1;S=S.prevZ}for(;x&&x.z<=E;){if(x.x>=d&&x.x<=y&&x.y>=p&&x.y<=m&&x!==s&&x!==o&&Ur(a,h,c,u,l,f,x.x,x.y)&&Et(x.prev,x,x.next)>=0)return!1;x=x.nextZ}return!0}function jp(i,e){let t=i;do{let n=t.prev,s=t.next.next;!sr(n,s)&&Bf(n,t,t.next,s)&&ro(n,s)&&ro(s,n)&&(e.push(n.i,t.i,s.i),oo(t),oo(t.next),t=i=s),t=t.next}while(t!==i);return gs(t)}function Jp(i,e,t,n,s,r){let o=i;do{let a=o.next.next;for(;a!==o.prev;){if(o.i!==a.i&&om(o,a)){let c=kf(o,a);o=gs(o,o.next),c=gs(c,c.next),so(o,e,t,n,s,r,0),so(c,e,t,n,s,r,0);return}a=a.next}o=o.next}while(o!==i)}function $p(i,e,t,n){let s=[];for(let r=0,o=e.length;r<o;r++){let a=e[r]*n,c=r<o-1?e[r+1]*n:i.length,l=Of(i,a,c,n,!1);l===l.next&&(l.steiner=!0),s.push(rm(l))}s.sort(Qp);for(let r=0;r<s.length;r++)t=em(s[r],t);return t}function Qp(i,e){let t=i.x-e.x;if(t===0&&(t=i.y-e.y,t===0)){let n=(i.next.y-i.y)/(i.next.x-i.x),s=(e.next.y-e.y)/(e.next.x-e.x);t=n-s}return t}function em(i,e){let t=tm(i,e);if(!t)return e;let n=kf(t,i);return gs(n,n.next),gs(t,t.next)}function tm(i,e){let t=e,n=i.x,s=i.y,r=-1/0,o;if(sr(i,t))return t;do{if(sr(i,t.next))return t.next;if(s<=t.y&&s>=t.next.y&&t.next.y!==t.y){let u=t.x+(s-t.y)*(t.next.x-t.x)/(t.next.y-t.y);if(u<=n&&u>r&&(r=u,o=t.x<t.next.x?t:t.next,u===n))return o}t=t.next}while(t!==e);if(!o)return null;let a=o,c=o.x,l=o.y,h=1/0;t=o;do{if(n>=t.x&&t.x>=c&&n!==t.x&&Ff(s<l?n:r,s,c,l,s<l?r:n,s,t.x,t.y)){let u=Math.abs(s-t.y)/(n-t.x);ro(t,i)&&(u<h||u===h&&(t.x>o.x||t.x===o.x&&nm(o,t)))&&(o=t,h=u)}t=t.next}while(t!==a);return o}function nm(i,e){return Et(i.prev,i,e.prev)<0&&Et(e.next,i,i.next)<0}function im(i,e,t,n){let s=i;do s.z===0&&(s.z=Sl(s.x,s.y,e,t,n)),s.prevZ=s.prev,s.nextZ=s.next,s=s.next;while(s!==i);s.prevZ.nextZ=null,s.prevZ=null,sm(s)}function sm(i){let e,t=1;do{let n=i,s;i=null;let r=null;for(e=0;n;){e++;let o=n,a=0;for(let l=0;l<t&&(a++,o=o.nextZ,!!o);l++);let c=t;for(;a>0||c>0&&o;)a!==0&&(c===0||!o||n.z<=o.z)?(s=n,n=n.nextZ,a--):(s=o,o=o.nextZ,c--),r?r.nextZ=s:i=s,s.prevZ=r,r=s;n=o}r.nextZ=null,t*=2}while(e>1);return i}function Sl(i,e,t,n,s){return i=(i-t)*s|0,e=(e-n)*s|0,i=(i|i<<8)&16711935,i=(i|i<<4)&252645135,i=(i|i<<2)&858993459,i=(i|i<<1)&1431655765,e=(e|e<<8)&16711935,e=(e|e<<4)&252645135,e=(e|e<<2)&858993459,e=(e|e<<1)&1431655765,i|e<<1}function rm(i){let e=i,t=i;do(e.x<t.x||e.x===t.x&&e.y<t.y)&&(t=e),e=e.next;while(e!==i);return t}function Ff(i,e,t,n,s,r,o,a){return(s-o)*(e-a)>=(i-o)*(r-a)&&(i-o)*(n-a)>=(t-o)*(e-a)&&(t-o)*(r-a)>=(s-o)*(n-a)}function Ur(i,e,t,n,s,r,o,a){return!(i===o&&e===a)&&Ff(i,e,t,n,s,r,o,a)}function om(i,e){return i.next.i!==e.i&&i.prev.i!==e.i&&!am(i,e)&&(ro(i,e)&&ro(e,i)&&cm(i,e)&&(Et(i.prev,i,e.prev)||Et(i,e.prev,e))||sr(i,e)&&Et(i.prev,i,i.next)>0&&Et(e.prev,e,e.next)>0)}function Et(i,e,t){return(e.y-i.y)*(t.x-e.x)-(e.x-i.x)*(t.y-e.y)}function sr(i,e){return i.x===e.x&&i.y===e.y}function Bf(i,e,t,n){let s=ta(Et(i,e,t)),r=ta(Et(i,e,n)),o=ta(Et(t,n,i)),a=ta(Et(t,n,e));return!!(s!==r&&o!==a||s===0&&ea(i,t,e)||r===0&&ea(i,n,e)||o===0&&ea(t,i,n)||a===0&&ea(t,e,n))}function ea(i,e,t){return e.x<=Math.max(i.x,t.x)&&e.x>=Math.min(i.x,t.x)&&e.y<=Math.max(i.y,t.y)&&e.y>=Math.min(i.y,t.y)}function ta(i){return i>0?1:i<0?-1:0}function am(i,e){let t=i;do{if(t.i!==i.i&&t.next.i!==i.i&&t.i!==e.i&&t.next.i!==e.i&&Bf(t,t.next,i,e))return!0;t=t.next}while(t!==i);return!1}function ro(i,e){return Et(i.prev,i,i.next)<0?Et(i,e,i.next)>=0&&Et(i,i.prev,e)>=0:Et(i,e,i.prev)<0||Et(i,i.next,e)<0}function cm(i,e){let t=i,n=!1,s=(i.x+e.x)/2,r=(i.y+e.y)/2;do t.y>r!=t.next.y>r&&t.next.y!==t.y&&s<(t.next.x-t.x)*(r-t.y)/(t.next.y-t.y)+t.x&&(n=!n),t=t.next;while(t!==i);return n}function kf(i,e){let t=Tl(i.i,i.x,i.y),n=Tl(e.i,e.x,e.y),s=i.next,r=e.prev;return i.next=e,e.prev=i,t.next=s,s.prev=t,n.next=t,t.prev=n,r.next=n,n.prev=r,n}function zu(i,e,t,n){let s=Tl(i,e,t);return n?(s.next=n.next,s.prev=n,n.next.prev=s,n.next=s):(s.prev=s,s.next=s),s}function oo(i){i.next.prev=i.prev,i.prev.next=i.next,i.prevZ&&(i.prevZ.nextZ=i.nextZ),i.nextZ&&(i.nextZ.prevZ=i.prevZ)}function Tl(i,e,t){return{i,x:e,y:t,prev:null,next:null,z:0,prevZ:null,nextZ:null,steiner:!1}}function lm(i,e,t,n){let s=0;for(let r=e,o=t-n;r<t;r+=n)s+=(i[o]-i[r])*(i[r+1]+i[o+1]),o=r;return s}var El=class{static triangulate(e,t,n=2){return Yp(e,t,n)}},Kn=class i{static area(e){let t=e.length,n=0;for(let s=t-1,r=0;r<t;s=r++)n+=e[s].x*e[r].y-e[r].x*e[s].y;return n*.5}static isClockWise(e){return i.area(e)<0}static triangulateShape(e,t){let n=[],s=[],r=[];Hu(e),Vu(n,e);let o=e.length;t.forEach(Hu);for(let c=0;c<t.length;c++)s.push(o),o+=t[c].length,Vu(n,t[c]);let a=El.triangulate(n,s);for(let c=0;c<a.length;c+=3)r.push(a.slice(c,c+3));return r}};function Hu(i){let e=i.length;e>2&&i[e-1].equals(i[0])&&i.pop()}function Vu(i,e){for(let t=0;t<e.length;t++)i.push(e[t].x),i.push(e[t].y)}var _s=class i extends wt{constructor(e=new kn([new de(.5,.5),new de(-.5,.5),new de(-.5,-.5),new de(.5,-.5)]),t={}){super(),this.type="ExtrudeGeometry",this.parameters={shapes:e,options:t},e=Array.isArray(e)?e:[e];let n=this,s=[],r=[];for(let a=0,c=e.length;a<c;a++){let l=e[a];o(l)}this.setAttribute("position",new Mt(s,3)),this.setAttribute("uv",new Mt(r,2)),this.computeVertexNormals();function o(a){let c=[],l=t.curveSegments!==void 0?t.curveSegments:12,h=t.steps!==void 0?t.steps:1,u=t.depth!==void 0?t.depth:1,f=t.bevelEnabled!==void 0?t.bevelEnabled:!0,d=t.bevelThickness!==void 0?t.bevelThickness:.2,p=t.bevelSize!==void 0?t.bevelSize:d-.1,y=t.bevelOffset!==void 0?t.bevelOffset:0,m=t.bevelSegments!==void 0?t.bevelSegments:3,g=t.extrudePath,E=t.UVGenerator!==void 0?t.UVGenerator:hm,S,x=!1,A,w,I,v;if(g){S=g.getSpacedPoints(h),x=!0,f=!1;let C=g.isCatmullRomCurve3?g.closed:!1;A=g.computeFrenetFrames(h,C),w=new B,I=new B,v=new B}f||(m=0,d=0,p=0,y=0);let R=a.extractPoints(l),F=R.shape,N=R.holes;if(!Kn.isClockWise(F)){F=F.reverse();for(let C=0,V=N.length;C<V;C++){let D=N[C];Kn.isClockWise(D)&&(N[C]=D.reverse())}}function oe(C){let D=10000000000000001e-36,j=C[0];for(let $=1;$<=C.length;$++){let le=$%C.length,te=C[le],ne=te.x-j.x,k=te.y-j.y,b=ne*ne+k*k,we=Math.max(Math.abs(te.x),Math.abs(te.y),Math.abs(j.x),Math.abs(j.y)),Ae=D*we*we;if(b<=Ae){C.splice(le,1),$--;continue}j=te}}oe(F),N.forEach(oe);let ie=N.length,X=F;for(let C=0;C<ie;C++){let V=N[C];F=F.concat(V)}function ee(C,V,D){return V||Ke("ExtrudeGeometry: vec does not exist"),C.clone().addScaledVector(V,D)}let Y=F.length;function G(C,V,D){let j,$,le,te=C.x-V.x,ne=C.y-V.y,k=D.x-C.x,b=D.y-C.y,we=te*te+ne*ne,Ae=te*b-ne*k;if(Math.abs(Ae)>Number.EPSILON){let T=Math.sqrt(we),_=Math.sqrt(k*k+b*b),O=V.x-ne/T,W=V.y+te/T,Q=D.x-b/_,_e=D.y+k/_,Se=((Q-O)*b-(_e-W)*k)/(te*b-ne*k);j=O+te*Se-C.x,$=W+ne*Se-C.y;let ae=j*j+$*$;if(ae<=2)return new de(j,$);le=Math.sqrt(ae/2)}else{let T=!1;te>Number.EPSILON?k>Number.EPSILON&&(T=!0):te<-Number.EPSILON?k<-Number.EPSILON&&(T=!0):Math.sign(ne)===Math.sign(b)&&(T=!0),T?(j=-ne,$=te,le=Math.sqrt(we)):(j=te,$=ne,le=Math.sqrt(we/2))}return new de(j/le,$/le)}let ge=[];for(let C=0,V=X.length,D=V-1,j=C+1;C<V;C++,D++,j++)D===V&&(D=0),j===V&&(j=0),ge[C]=G(X[C],X[D],X[j]);let ye=[],Me,Re=ge.concat();for(let C=0,V=ie;C<V;C++){let D=N[C];Me=[];for(let j=0,$=D.length,le=$-1,te=j+1;j<$;j++,le++,te++)le===$&&(le=0),te===$&&(te=0),Me[j]=G(D[j],D[le],D[te]);ye.push(Me),Re=Re.concat(Me)}let ze;if(m===0)ze=Kn.triangulateShape(X,N);else{let C=[],V=[];for(let D=0;D<m;D++){let j=D/m,$=d*Math.cos(j*Math.PI/2),le=p*Math.sin(j*Math.PI/2)+y;for(let te=0,ne=X.length;te<ne;te++){let k=ee(X[te],ge[te],le);L(k.x,k.y,-$),j===0&&C.push(k)}for(let te=0,ne=ie;te<ne;te++){let k=N[te];Me=ye[te];let b=[];for(let we=0,Ae=k.length;we<Ae;we++){let T=ee(k[we],Me[we],le);L(T.x,T.y,-$),j===0&&b.push(T)}j===0&&V.push(b)}}ze=Kn.triangulateShape(C,V)}let qe=ze.length,We=p+y;for(let C=0;C<Y;C++){let V=f?ee(F[C],Re[C],We):F[C];x?(I.copy(A.normals[0]).multiplyScalar(V.x),w.copy(A.binormals[0]).multiplyScalar(V.y),v.copy(S[0]).add(I).add(w),L(v.x,v.y,v.z)):L(V.x,V.y,0)}for(let C=1;C<=h;C++)for(let V=0;V<Y;V++){let D=f?ee(F[V],Re[V],We):F[V];x?(I.copy(A.normals[C]).multiplyScalar(D.x),w.copy(A.binormals[C]).multiplyScalar(D.y),v.copy(S[C]).add(I).add(w),L(v.x,v.y,v.z)):L(D.x,D.y,u/h*C)}for(let C=m-1;C>=0;C--){let V=C/m,D=d*Math.cos(V*Math.PI/2),j=p*Math.sin(V*Math.PI/2)+y;for(let $=0,le=X.length;$<le;$++){let te=ee(X[$],ge[$],j);L(te.x,te.y,u+D)}for(let $=0,le=N.length;$<le;$++){let te=N[$];Me=ye[$];for(let ne=0,k=te.length;ne<k;ne++){let b=ee(te[ne],Me[ne],j);x?L(b.x,b.y+S[h-1].y,S[h-1].x+D):L(b.x,b.y,u+D)}}}pe(),q();function pe(){let C=s.length/3;if(f){let V=0,D=Y*V;for(let j=0;j<qe;j++){let $=ze[j];P($[2]+D,$[1]+D,$[0]+D)}V=h+m*2,D=Y*V;for(let j=0;j<qe;j++){let $=ze[j];P($[0]+D,$[1]+D,$[2]+D)}}else{for(let V=0;V<qe;V++){let D=ze[V];P(D[2],D[1],D[0])}for(let V=0;V<qe;V++){let D=ze[V];P(D[0]+Y*h,D[1]+Y*h,D[2]+Y*h)}}n.addGroup(C,s.length/3-C,0)}function q(){let C=s.length/3,V=0;U(X,V),V+=X.length;for(let D=0,j=N.length;D<j;D++){let $=N[D];U($,V),V+=$.length}n.addGroup(C,s.length/3-C,1)}function U(C,V){let D=C.length;for(;--D>=0;){let j=D,$=D-1;$<0&&($=C.length-1);for(let le=0,te=h+m*2;le<te;le++){let ne=Y*le,k=Y*(le+1),b=V+j+ne,we=V+$+ne,Ae=V+$+k,T=V+j+k;z(b,we,Ae,T)}}}function L(C,V,D){c.push(C),c.push(V),c.push(D)}function P(C,V,D){he(C),he(V),he(D);let j=s.length/3,$=E.generateTopUV(n,s,j-3,j-2,j-1);me($[0]),me($[1]),me($[2])}function z(C,V,D,j){he(C),he(V),he(j),he(V),he(D),he(j);let $=s.length/3,le=E.generateSideWallUV(n,s,$-6,$-3,$-2,$-1);me(le[0]),me(le[1]),me(le[3]),me(le[1]),me(le[2]),me(le[3])}function he(C){s.push(c[C*3+0]),s.push(c[C*3+1]),s.push(c[C*3+2])}function me(C){r.push(C.x),r.push(C.y)}}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}toJSON(){let e=super.toJSON(),t=this.parameters.shapes,n=this.parameters.options;return um(t,n,e)}static fromJSON(e,t){let n=[];for(let r=0,o=e.shapes.length;r<o;r++){let a=t[e.shapes[r]];n.push(a)}let s=e.options.extrudePath;return s!==void 0&&(e.options.extrudePath=new Ml[s.type]().fromJSON(s)),new i(n,e.options)}},hm={generateTopUV:function(i,e,t,n,s){let r=e[t*3],o=e[t*3+1],a=e[n*3],c=e[n*3+1],l=e[s*3],h=e[s*3+1];return[new de(r,o),new de(a,c),new de(l,h)]},generateSideWallUV:function(i,e,t,n,s,r){let o=e[t*3],a=e[t*3+1],c=e[t*3+2],l=e[n*3],h=e[n*3+1],u=e[n*3+2],f=e[s*3],d=e[s*3+1],p=e[s*3+2],y=e[r*3],m=e[r*3+1],g=e[r*3+2];return Math.abs(a-h)<Math.abs(o-l)?[new de(o,1-c),new de(l,1-u),new de(f,1-p),new de(y,1-g)]:[new de(a,1-c),new de(h,1-u),new de(d,1-p),new de(m,1-g)]}};function um(i,e,t){if(t.shapes=[],Array.isArray(i))for(let n=0,s=i.length;n<s;n++){let r=i[n];t.shapes.push(r.uuid)}else t.shapes.push(i.uuid);return t.options=Object.assign({},e),e.extrudePath!==void 0&&(t.options.extrudePath=e.extrudePath.toJSON()),t}var xs=class i extends wt{constructor(e=1,t=1,n=1,s=1){super(),this.type="PlaneGeometry",this.parameters={width:e,height:t,widthSegments:n,heightSegments:s};let r=e/2,o=t/2,a=Math.floor(n),c=Math.floor(s),l=a+1,h=c+1,u=e/a,f=t/c,d=[],p=[],y=[],m=[];for(let g=0;g<h;g++){let E=g*f-o;for(let S=0;S<l;S++){let x=S*u-r;p.push(x,-E,0),y.push(0,0,1),m.push(S/a),m.push(1-g/c)}}for(let g=0;g<c;g++)for(let E=0;E<a;E++){let S=E+l*g,x=E+l*(g+1),A=E+1+l*(g+1),w=E+1+l*g;d.push(S,x,w),d.push(x,A,w)}this.setIndex(d),this.setAttribute("position",new Mt(p,3)),this.setAttribute("normal",new Mt(y,3)),this.setAttribute("uv",new Mt(m,2))}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new i(e.width,e.height,e.widthSegments,e.heightSegments)}},ys=class i extends wt{constructor(e=.5,t=1,n=32,s=1,r=0,o=Math.PI*2){super(),this.type="RingGeometry",this.parameters={innerRadius:e,outerRadius:t,thetaSegments:n,phiSegments:s,thetaStart:r,thetaLength:o},n=Math.max(3,n),s=Math.max(1,s);let a=[],c=[],l=[],h=[],u=e,f=(t-e)/s,d=new B,p=new de;for(let y=0;y<=s;y++){for(let m=0;m<=n;m++){let g=r+m/n*o;d.x=u*Math.cos(g),d.y=u*Math.sin(g),c.push(d.x,d.y,d.z),l.push(0,0,1),p.x=(d.x/t+1)/2,p.y=(d.y/t+1)/2,h.push(p.x,p.y)}u+=f}for(let y=0;y<s;y++){let m=y*(n+1);for(let g=0;g<n;g++){let E=g+m,S=E,x=E+n+1,A=E+n+2,w=E+1;a.push(S,x,w),a.push(x,A,w)}}this.setIndex(a),this.setAttribute("position",new Mt(c,3)),this.setAttribute("normal",new Mt(l,3)),this.setAttribute("uv",new Mt(h,2))}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new i(e.innerRadius,e.outerRadius,e.thetaSegments,e.phiSegments,e.thetaStart,e.thetaLength)}};function Es(i){let e={};for(let t in i){e[t]={};for(let n in i[t]){let s=i[t][n];if(Gu(s))s.isRenderTargetTexture?(ke("UniformsUtils: Textures of render targets cannot be cloned via cloneUniforms() or mergeUniforms()."),e[t][n]=null):e[t][n]=s.clone();else if(Array.isArray(s))if(Gu(s[0])){let r=[];for(let o=0,a=s.length;o<a;o++)r[o]=s[o].clone();e[t][n]=r}else e[t][n]=s.slice();else e[t][n]=s}}return e}function Qt(i){let e={};for(let t=0;t<i.length;t++){let n=Es(i[t]);for(let s in n)e[s]=n[s]}return e}function Gu(i){return i&&(i.isColor||i.isMatrix3||i.isMatrix4||i.isVector2||i.isVector3||i.isVector4||i.isTexture||i.isQuaternion)}function fm(i){let e=[];for(let t=0;t<i.length;t++)e.push(i[t].clone());return e}function Ql(i){let e=i.getRenderTarget();return e===null?i.outputColorSpace:e.isXRRenderTarget===!0?e.texture.colorSpace:Qe.workingColorSpace}var zf={clone:Es,merge:Qt},dm=`void main() {
	gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
}`,pm=`void main() {
	gl_FragColor = vec4( 1.0, 0.0, 0.0, 1.0 );
}`,Jt=class extends on{constructor(e){super(),this.isShaderMaterial=!0,this.type="ShaderMaterial",this.defines={},this.uniforms={},this.uniformsGroups=[],this.vertexShader=dm,this.fragmentShader=pm,this.linewidth=1,this.wireframe=!1,this.wireframeLinewidth=1,this.fog=!1,this.lights=!1,this.clipping=!1,this.forceSinglePass=!0,this.extensions={clipCullDistance:!1,multiDraw:!1},this.defaultAttributeValues={color:[1,1,1],uv:[0,0],uv1:[0,0]},this.index0AttributeName=void 0,this.uniformsNeedUpdate=!1,this.glslVersion=null,e!==void 0&&this.setValues(e)}copy(e){return super.copy(e),this.fragmentShader=e.fragmentShader,this.vertexShader=e.vertexShader,this.uniforms=Es(e.uniforms),this.uniformsGroups=fm(e.uniformsGroups),this.defines=Object.assign({},e.defines),this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.fog=e.fog,this.lights=e.lights,this.clipping=e.clipping,this.extensions=Object.assign({},e.extensions),this.glslVersion=e.glslVersion,this.defaultAttributeValues=Object.assign({},e.defaultAttributeValues),this.index0AttributeName=e.index0AttributeName,this.uniformsNeedUpdate=e.uniformsNeedUpdate,this}toJSON(e){let t=super.toJSON(e);t.glslVersion=this.glslVersion,t.uniforms={};for(let s in this.uniforms){let o=this.uniforms[s].value;o&&o.isTexture?t.uniforms[s]={type:"t",value:o.toJSON(e).uuid}:o&&o.isColor?t.uniforms[s]={type:"c",value:o.getHex()}:o&&o.isVector2?t.uniforms[s]={type:"v2",value:o.toArray()}:o&&o.isVector3?t.uniforms[s]={type:"v3",value:o.toArray()}:o&&o.isVector4?t.uniforms[s]={type:"v4",value:o.toArray()}:o&&o.isMatrix3?t.uniforms[s]={type:"m3",value:o.toArray()}:o&&o.isMatrix4?t.uniforms[s]={type:"m4",value:o.toArray()}:t.uniforms[s]={value:o}}Object.keys(this.defines).length>0&&(t.defines=this.defines),t.vertexShader=this.vertexShader,t.fragmentShader=this.fragmentShader,t.lights=this.lights,t.clipping=this.clipping;let n={};for(let s in this.extensions)this.extensions[s]===!0&&(n[s]=!0);return Object.keys(n).length>0&&(t.extensions=n),t}fromJSON(e,t){if(super.fromJSON(e,t),e.uniforms!==void 0)for(let n in e.uniforms){let s=e.uniforms[n];switch(this.uniforms[n]={},s.type){case"t":this.uniforms[n].value=t[s.value]||null;break;case"c":this.uniforms[n].value=new Ge().setHex(s.value);break;case"v2":this.uniforms[n].value=new de().fromArray(s.value);break;case"v3":this.uniforms[n].value=new B().fromArray(s.value);break;case"v4":this.uniforms[n].value=new ft().fromArray(s.value);break;case"m3":this.uniforms[n].value=new Ze().fromArray(s.value);break;case"m4":this.uniforms[n].value=new Je().fromArray(s.value);break;default:this.uniforms[n].value=s.value}}if(e.defines!==void 0&&(this.defines=e.defines),e.vertexShader!==void 0&&(this.vertexShader=e.vertexShader),e.fragmentShader!==void 0&&(this.fragmentShader=e.fragmentShader),e.glslVersion!==void 0&&(this.glslVersion=e.glslVersion),e.extensions!==void 0)for(let n in e.extensions)this.extensions[n]=e.extensions[n];return e.lights!==void 0&&(this.lights=e.lights),e.clipping!==void 0&&(this.clipping=e.clipping),this}},Ra=class extends Jt{constructor(e){super(e),this.isRawShaderMaterial=!0,this.type="RawShaderMaterial"}},an=class extends on{constructor(e){super(),this.isMeshStandardMaterial=!0,this.type="MeshStandardMaterial",this.defines={STANDARD:""},this.color=new Ge(16777215),this.roughness=1,this.metalness=0,this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.emissive=new Ge(0),this.emissiveIntensity=1,this.emissiveMap=null,this.bumpMap=null,this.bumpScale=1,this.normalMap=null,this.normalMapType=Ec,this.normalScale=new de(1,1),this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.roughnessMap=null,this.metalnessMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new gi,this.envMapIntensity=1,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.flatShading=!1,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.defines={STANDARD:""},this.color.copy(e.color),this.roughness=e.roughness,this.metalness=e.metalness,this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.emissive.copy(e.emissive),this.emissiveMap=e.emissiveMap,this.emissiveIntensity=e.emissiveIntensity,this.bumpMap=e.bumpMap,this.bumpScale=e.bumpScale,this.normalMap=e.normalMap,this.normalMapType=e.normalMapType,this.normalScale.copy(e.normalScale),this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.roughnessMap=e.roughnessMap,this.metalnessMap=e.metalnessMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.envMapIntensity=e.envMapIntensity,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.flatShading=e.flatShading,this.fog=e.fog,this}},$t=class extends an{constructor(e){super(),this.isMeshPhysicalMaterial=!0,this.defines={STANDARD:"",PHYSICAL:""},this.type="MeshPhysicalMaterial",this.anisotropyRotation=0,this.anisotropyMap=null,this.clearcoatMap=null,this.clearcoatRoughness=0,this.clearcoatRoughnessMap=null,this.clearcoatNormalScale=new de(1,1),this.clearcoatNormalMap=null,this.ior=1.5,Object.defineProperty(this,"reflectivity",{get:function(){return et(2.5*(this.ior-1)/(this.ior+1),0,1)},set:function(t){this.ior=(1+.4*t)/(1-.4*t)}}),this.iridescenceMap=null,this.iridescenceIOR=1.3,this.iridescenceThicknessRange=[100,400],this.iridescenceThicknessMap=null,this.sheenColor=new Ge(0),this.sheenColorMap=null,this.sheenRoughness=1,this.sheenRoughnessMap=null,this.transmissionMap=null,this.thickness=0,this.thicknessMap=null,this.attenuationDistance=1/0,this.attenuationColor=new Ge(1,1,1),this.specularIntensity=1,this.specularIntensityMap=null,this.specularColor=new Ge(1,1,1),this.specularColorMap=null,this._anisotropy=0,this._clearcoat=0,this._dispersion=0,this._iridescence=0,this._sheen=0,this._transmission=0,this.setValues(e)}get anisotropy(){return this._anisotropy}set anisotropy(e){this._anisotropy>0!=e>0&&this.version++,this._anisotropy=e}get clearcoat(){return this._clearcoat}set clearcoat(e){this._clearcoat>0!=e>0&&this.version++,this._clearcoat=e}get iridescence(){return this._iridescence}set iridescence(e){this._iridescence>0!=e>0&&this.version++,this._iridescence=e}get dispersion(){return this._dispersion}set dispersion(e){this._dispersion>0!=e>0&&this.version++,this._dispersion=e}get sheen(){return this._sheen}set sheen(e){this._sheen>0!=e>0&&this.version++,this._sheen=e}get transmission(){return this._transmission}set transmission(e){this._transmission>0!=e>0&&this.version++,this._transmission=e}copy(e){return super.copy(e),this.defines={STANDARD:"",PHYSICAL:""},this.anisotropy=e.anisotropy,this.anisotropyRotation=e.anisotropyRotation,this.anisotropyMap=e.anisotropyMap,this.clearcoat=e.clearcoat,this.clearcoatMap=e.clearcoatMap,this.clearcoatRoughness=e.clearcoatRoughness,this.clearcoatRoughnessMap=e.clearcoatRoughnessMap,this.clearcoatNormalMap=e.clearcoatNormalMap,this.clearcoatNormalScale.copy(e.clearcoatNormalScale),this.dispersion=e.dispersion,this.ior=e.ior,this.iridescence=e.iridescence,this.iridescenceMap=e.iridescenceMap,this.iridescenceIOR=e.iridescenceIOR,this.iridescenceThicknessRange=[...e.iridescenceThicknessRange],this.iridescenceThicknessMap=e.iridescenceThicknessMap,this.sheen=e.sheen,this.sheenColor.copy(e.sheenColor),this.sheenColorMap=e.sheenColorMap,this.sheenRoughness=e.sheenRoughness,this.sheenRoughnessMap=e.sheenRoughnessMap,this.transmission=e.transmission,this.transmissionMap=e.transmissionMap,this.thickness=e.thickness,this.thicknessMap=e.thicknessMap,this.attenuationDistance=e.attenuationDistance,this.attenuationColor.copy(e.attenuationColor),this.specularIntensity=e.specularIntensity,this.specularIntensityMap=e.specularIntensityMap,this.specularColor.copy(e.specularColor),this.specularColorMap=e.specularColorMap,this}};var Ca=class extends on{constructor(e){super(),this.isMeshDepthMaterial=!0,this.type="MeshDepthMaterial",this.depthPacking=Sf,this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.wireframe=!1,this.wireframeLinewidth=1,this.setValues(e)}copy(e){return super.copy(e),this.depthPacking=e.depthPacking,this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this}},Pa=class extends on{constructor(e){super(),this.isMeshDistanceMaterial=!0,this.type="MeshDistanceMaterial",this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.setValues(e)}copy(e){return super.copy(e),this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this}};function na(i,e){return!i||i.constructor===e?i:typeof e.BYTES_PER_ELEMENT=="number"?new e(i):Array.prototype.slice.call(i)}function mm(i){function e(s,r){return i[s]-i[r]}let t=i.length,n=new Array(t);for(let s=0;s!==t;++s)n[s]=s;return n.sort(e),n}function Wu(i,e,t){let n=i.length,s=new i.constructor(n);for(let r=0,o=0;o!==n;++r){let a=t[r]*e;for(let c=0;c!==e;++c)s[o++]=i[a+c]}return s}function gm(i,e,t,n){let s=1,r=i[0];for(;r!==void 0&&r[n]===void 0;)r=i[s++];if(r===void 0)return;let o=r[n];if(o!==void 0)if(Array.isArray(o))do o=r[n],o!==void 0&&(e.push(r.time),t.push(...o)),r=i[s++];while(r!==void 0);else if(o.toArray!==void 0)do o=r[n],o!==void 0&&(e.push(r.time),o.toArray(t,t.length)),r=i[s++];while(r!==void 0);else do o=r[n],o!==void 0&&(e.push(r.time),t.push(o)),r=i[s++];while(r!==void 0)}var ei=class{constructor(e,t,n,s){this.parameterPositions=e,this._cachedIndex=0,this.resultBuffer=s!==void 0?s:new t.constructor(n),this.sampleValues=t,this.valueSize=n,this.settings=null,this.DefaultSettings_={}}evaluate(e){let t=this.parameterPositions,n=this._cachedIndex,s=t[n],r=t[n-1];n:{e:{let o;t:{i:if(!(e<s)){for(let a=n+2;;){if(s===void 0){if(e<r)break i;return n=t.length,this._cachedIndex=n,this.copySampleValue_(n-1)}if(n===a)break;if(r=s,s=t[++n],e<s)break e}o=t.length;break t}if(!(e>=r)){let a=t[1];e<a&&(n=2,r=a);for(let c=n-2;;){if(r===void 0)return this._cachedIndex=0,this.copySampleValue_(0);if(n===c)break;if(s=r,r=t[--n-1],e>=r)break e}o=n,n=0;break t}break n}for(;n<o;){let a=n+o>>>1;e<t[a]?o=a:n=a+1}if(s=t[n],r=t[n-1],r===void 0)return this._cachedIndex=0,this.copySampleValue_(0);if(s===void 0)return n=t.length,this._cachedIndex=n,this.copySampleValue_(n-1)}this._cachedIndex=n,this.intervalChanged_(n,r,s)}return this.interpolate_(n,r,e,s)}getSettings_(){return this.settings||this.DefaultSettings_}copySampleValue_(e){let t=this.resultBuffer,n=this.sampleValues,s=this.valueSize,r=e*s;for(let o=0;o!==s;++o)t[o]=n[r+o];return t}interpolate_(){throw new Error("THREE.Interpolant: Call to abstract method.")}intervalChanged_(){}},Ia=class extends ei{constructor(e,t,n,s){super(e,t,n,s),this._weightPrev=-0,this._offsetPrev=-0,this._weightNext=-0,this._offsetNext=-0,this.DefaultSettings_={endingStart:_l,endingEnd:_l}}intervalChanged_(e,t,n){let s=this.parameterPositions,r=e-2,o=e+1,a=s[r],c=s[o];if(a===void 0)switch(this.getSettings_().endingStart){case xl:r=e,a=2*t-n;break;case yl:r=s.length-2,a=t+s[r]-s[r+1];break;default:r=e,a=n}if(c===void 0)switch(this.getSettings_().endingEnd){case xl:o=e,c=2*n-t;break;case yl:o=1,c=n+s[1]-s[0];break;default:o=e-1,c=t}let l=(n-t)*.5,h=this.valueSize;this._weightPrev=l/(t-a),this._weightNext=l/(c-n),this._offsetPrev=r*h,this._offsetNext=o*h}interpolate_(e,t,n,s){let r=this.resultBuffer,o=this.sampleValues,a=this.valueSize,c=e*a,l=c-a,h=this._offsetPrev,u=this._offsetNext,f=this._weightPrev,d=this._weightNext,p=(n-t)/(s-t),y=p*p,m=y*p,g=-f*m+2*f*y-f*p,E=(1+f)*m+(-1.5-2*f)*y+(-.5+f)*p+1,S=(-1-d)*m+(1.5+d)*y+.5*p,x=d*m-d*y;for(let A=0;A!==a;++A)r[A]=g*o[h+A]+E*o[l+A]+S*o[c+A]+x*o[u+A];return r}},La=class extends ei{constructor(e,t,n,s){super(e,t,n,s)}interpolate_(e,t,n,s){let r=this.resultBuffer,o=this.sampleValues,a=this.valueSize,c=e*a,l=c-a,h=(n-t)/(s-t),u=1-h;for(let f=0;f!==a;++f)r[f]=o[l+f]*u+o[c+f]*h;return r}},Da=class extends ei{constructor(e,t,n,s){super(e,t,n,s)}interpolate_(e){return this.copySampleValue_(e-1)}},Na=class extends ei{interpolate_(e,t,n,s){let r=this.resultBuffer,o=this.sampleValues,a=this.valueSize,c=e*a,l=c-a,h=this.inTangents,u=this.outTangents;if(!h||!u){let p=(n-t)/(s-t),y=1-p;for(let m=0;m!==a;++m)r[m]=o[l+m]*y+o[c+m]*p;return r}let f=a*2,d=e-1;for(let p=0;p!==a;++p){let y=o[l+p],m=o[c+p],g=d*f+p*2,E=u[g],S=u[g+1],x=e*f+p*2,A=h[x],w=h[x+1],I=(n-t)/(s-t),v,R,F,N,H;for(let oe=0;oe<8;oe++){v=I*I,R=v*I,F=1-I,N=F*F,H=N*F;let X=H*t+3*N*I*E+3*F*v*A+R*s-n;if(Math.abs(X)<1e-10)break;let ee=3*N*(E-t)+6*F*I*(A-E)+3*v*(s-A);if(Math.abs(ee)<1e-10)break;I=I-X/ee,I=Math.max(0,Math.min(1,I))}r[p]=H*y+3*N*I*S+3*F*v*w+R*m}return r}},cn=class{constructor(e,t,n,s){if(e===void 0)throw new Error("THREE.KeyframeTrack: track name is undefined");if(t===void 0||t.length===0)throw new Error("THREE.KeyframeTrack: no keyframes in track named "+e);this.name=e,this.times=na(t,this.TimeBufferType),this.values=na(n,this.ValueBufferType),this.setInterpolation(s||this.DefaultInterpolation)}static toJSON(e){let t=e.constructor,n;if(t.toJSON!==this.toJSON)n=t.toJSON(e);else{n={name:e.name,times:na(e.times,Array),values:na(e.values,Array)};let s=e.getInterpolation();s!==e.DefaultInterpolation&&(n.interpolation=s)}return n.type=e.ValueTypeName,n}InterpolantFactoryMethodDiscrete(e){return new Da(this.times,this.values,this.getValueSize(),e)}InterpolantFactoryMethodLinear(e){return new La(this.times,this.values,this.getValueSize(),e)}InterpolantFactoryMethodSmooth(e){return new Ia(this.times,this.values,this.getValueSize(),e)}InterpolantFactoryMethodBezier(e){let t=new Na(this.times,this.values,this.getValueSize(),e);return this.settings&&(t.inTangents=this.settings.inTangents,t.outTangents=this.settings.outTangents),t}setInterpolation(e){let t;switch(e){case fs:t=this.InterpolantFactoryMethodDiscrete;break;case ds:t=this.InterpolantFactoryMethodLinear;break;case ra:t=this.InterpolantFactoryMethodSmooth;break;case gl:t=this.InterpolantFactoryMethodBezier;break}if(t===void 0){let n="unsupported interpolation for "+this.ValueTypeName+" keyframe track named "+this.name;if(this.createInterpolant===void 0)if(e!==this.DefaultInterpolation)this.setInterpolation(this.DefaultInterpolation);else throw new Error(n);return ke("KeyframeTrack:",n),this}return this.createInterpolant=t,this}getInterpolation(){switch(this.createInterpolant){case this.InterpolantFactoryMethodDiscrete:return fs;case this.InterpolantFactoryMethodLinear:return ds;case this.InterpolantFactoryMethodSmooth:return ra;case this.InterpolantFactoryMethodBezier:return gl}}getValueSize(){return this.values.length/this.times.length}shift(e){if(e!==0){let t=this.times;for(let n=0,s=t.length;n!==s;++n)t[n]+=e}return this}scale(e){if(e!==1){let t=this.times;for(let n=0,s=t.length;n!==s;++n)t[n]*=e}return this}trim(e,t){let n=this.times,s=n.length,r=0,o=s-1;for(;r!==s&&n[r]<e;)++r;for(;o!==-1&&n[o]>t;)--o;if(++o,r!==0||o!==s){r>=o&&(o=Math.max(o,1),r=o-1);let a=this.getValueSize();this.times=n.slice(r,o),this.values=this.values.slice(r*a,o*a)}return this}validate(){let e=!0,t=this.getValueSize();t-Math.floor(t)!==0&&(Ke("KeyframeTrack: Invalid value size in track.",this),e=!1);let n=this.times,s=this.values,r=n.length;r===0&&(Ke("KeyframeTrack: Track is empty.",this),e=!1);let o=null;for(let a=0;a!==r;a++){let c=n[a];if(typeof c=="number"&&isNaN(c)){Ke("KeyframeTrack: Time is not a valid number.",this,a,c),e=!1;break}if(o!==null&&o>c){Ke("KeyframeTrack: Out of order keys.",this,a,c,o),e=!1;break}o=c}if(s!==void 0&&np(s))for(let a=0,c=s.length;a!==c;++a){let l=s[a];if(isNaN(l)){Ke("KeyframeTrack: Value is not a valid number.",this,a,l),e=!1;break}}return e}optimize(){let e=this.times.slice(),t=this.values.slice(),n=this.getValueSize(),s=this.getInterpolation()===ra,r=e.length-1,o=1;for(let a=1;a<r;++a){let c=!1,l=e[a],h=e[a+1];if(l!==h&&(a!==1||l!==e[0]))if(s)c=!0;else{let u=a*n,f=u-n,d=u+n;for(let p=0;p!==n;++p){let y=t[u+p];if(y!==t[f+p]||y!==t[d+p]){c=!0;break}}}if(c){if(a!==o){e[o]=e[a];let u=a*n,f=o*n;for(let d=0;d!==n;++d)t[f+d]=t[u+d]}++o}}if(r>0){e[o]=e[r];for(let a=r*n,c=o*n,l=0;l!==n;++l)t[c+l]=t[a+l];++o}return o!==e.length?(this.times=e.slice(0,o),this.values=t.slice(0,o*n)):(this.times=e,this.values=t),this}clone(){let e=this.times.slice(),t=this.values.slice(),n=this.constructor,s=new n(this.name,e,t);return s.createInterpolant=this.createInterpolant,s}};cn.prototype.ValueTypeName="";cn.prototype.TimeBufferType=Float32Array;cn.prototype.ValueBufferType=Float32Array;cn.prototype.DefaultInterpolation=ds;var yi=class extends cn{constructor(e,t,n){super(e,t,n)}};yi.prototype.ValueTypeName="bool";yi.prototype.ValueBufferType=Array;yi.prototype.DefaultInterpolation=fs;yi.prototype.InterpolantFactoryMethodLinear=void 0;yi.prototype.InterpolantFactoryMethodSmooth=void 0;var ao=class extends cn{constructor(e,t,n,s){super(e,t,n,s)}};ao.prototype.ValueTypeName="color";var vi=class extends cn{constructor(e,t,n,s){super(e,t,n,s)}};vi.prototype.ValueTypeName="number";var Ua=class extends ei{constructor(e,t,n,s){super(e,t,n,s)}interpolate_(e,t,n,s){let r=this.resultBuffer,o=this.sampleValues,a=this.valueSize,c=(n-t)/(s-t),l=e*a;for(let h=l+a;l!==h;l+=4)Kt.slerpFlat(r,0,o,l-a,o,l,c);return r}},bi=class extends cn{constructor(e,t,n,s){super(e,t,n,s)}InterpolantFactoryMethodLinear(e){return new Ua(this.times,this.values,this.getValueSize(),e)}};bi.prototype.ValueTypeName="quaternion";bi.prototype.InterpolantFactoryMethodSmooth=void 0;var Mi=class extends cn{constructor(e,t,n){super(e,t,n)}};Mi.prototype.ValueTypeName="string";Mi.prototype.ValueBufferType=Array;Mi.prototype.DefaultInterpolation=fs;Mi.prototype.InterpolantFactoryMethodLinear=void 0;Mi.prototype.InterpolantFactoryMethodSmooth=void 0;var Xi=class extends cn{constructor(e,t,n,s){super(e,t,n,s)}};Xi.prototype.ValueTypeName="vector";var co=class{constructor(e="",t=-1,n=[],s=Mf){this.name=e,this.tracks=n,this.duration=t,this.blendMode=s,this.uuid=Tn(),this.userData={},this.duration<0&&this.resetDuration()}static parse(e){let t=[],n=e.tracks,s=1/(e.fps||1);for(let o=0,a=n.length;o!==a;++o)t.push(xm(n[o]).scale(s));let r=new this(e.name,e.duration,t,e.blendMode);return r.uuid=e.uuid,r.userData=JSON.parse(e.userData||"{}"),r}static toJSON(e){let t=[],n=e.tracks,s={name:e.name,duration:e.duration,tracks:t,uuid:e.uuid,blendMode:e.blendMode,userData:JSON.stringify(e.userData)};for(let r=0,o=n.length;r!==o;++r)t.push(cn.toJSON(n[r]));return s}static CreateFromMorphTargetSequence(e,t,n,s){let r=t.length,o=[];for(let a=0;a<r;a++){let c=[],l=[];c.push((a+r-1)%r,a,(a+1)%r),l.push(0,1,0);let h=mm(c);c=Wu(c,1,h),l=Wu(l,1,h),!s&&c[0]===0&&(c.push(r),l.push(l[0])),o.push(new vi(".morphTargetInfluences["+t[a].name+"]",c,l).scale(1/n))}return new this(e,-1,o)}static findByName(e,t){let n=e;if(!Array.isArray(e)){let s=e;n=s.geometry&&s.geometry.animations||s.animations}for(let s=0;s<n.length;s++)if(n[s].name===t)return n[s];return null}static CreateClipsFromMorphTargetSequences(e,t,n){let s={},r=/^([\w-]*?)([\d]+)$/;for(let a=0,c=e.length;a<c;a++){let l=e[a],h=l.name.match(r);if(h&&h.length>1){let u=h[1],f=s[u];f||(s[u]=f=[]),f.push(l)}}let o=[];for(let a in s)o.push(this.CreateFromMorphTargetSequence(a,s[a],t,n));return o}resetDuration(){let e=this.tracks,t=0;for(let n=0,s=e.length;n!==s;++n){let r=this.tracks[n];t=Math.max(t,r.times[r.times.length-1])}return this.duration=t,this}trim(){for(let e=0;e<this.tracks.length;e++)this.tracks[e].trim(0,this.duration);return this}validate(){let e=!0;for(let t=0;t<this.tracks.length;t++)e=e&&this.tracks[t].validate();return e}optimize(){for(let e=0;e<this.tracks.length;e++)this.tracks[e].optimize();return this}clone(){let e=[];for(let n=0;n<this.tracks.length;n++)e.push(this.tracks[n].clone());let t=new this.constructor(this.name,this.duration,e,this.blendMode);return t.userData=JSON.parse(JSON.stringify(this.userData)),t}toJSON(){return this.constructor.toJSON(this)}};function _m(i){switch(i.toLowerCase()){case"scalar":case"double":case"float":case"number":case"integer":return vi;case"vector":case"vector2":case"vector3":case"vector4":return Xi;case"color":return ao;case"quaternion":return bi;case"bool":case"boolean":return yi;case"string":return Mi}throw new Error("THREE.KeyframeTrack: Unsupported typeName: "+i)}function xm(i){if(i.type===void 0)throw new Error("THREE.KeyframeTrack: track type undefined, can not parse");let e=_m(i.type);if(i.times===void 0){let t=[],n=[];gm(i.keys,t,n,"value"),i.times=t,i.values=n}return e.parse!==void 0?e.parse(i):new e(i.name,i.times,i.values,i.interpolation)}var jn={enabled:!1,files:{},add:function(i,e){this.enabled!==!1&&(Xu(i)||(this.files[i]=e))},get:function(i){if(this.enabled!==!1&&!Xu(i))return this.files[i]},remove:function(i){delete this.files[i]},clear:function(){this.files={}}};function Xu(i){try{let e=i.slice(i.indexOf(":")+1);return new URL(e).protocol==="blob:"}catch{return!1}}var Oa=class{constructor(e,t,n){let s=this,r=!1,o=0,a=0,c,l=[];this.onStart=void 0,this.onLoad=e,this.onProgress=t,this.onError=n,this._abortController=null,this.itemStart=function(h){a++,r===!1&&s.onStart!==void 0&&s.onStart(h,o,a),r=!0},this.itemEnd=function(h){o++,s.onProgress!==void 0&&s.onProgress(h,o,a),o===a&&(r=!1,s.onLoad!==void 0&&s.onLoad())},this.itemError=function(h){s.onError!==void 0&&s.onError(h)},this.resolveURL=function(h){return h=h.normalize("NFC"),c?c(h):h},this.setURLModifier=function(h){return c=h,this},this.addHandler=function(h,u){return l.push(h,u),this},this.removeHandler=function(h){let u=l.indexOf(h);return u!==-1&&l.splice(u,2),this},this.getHandler=function(h){for(let u=0,f=l.length;u<f;u+=2){let d=l[u],p=l[u+1];if(d.global&&(d.lastIndex=0),d.test(h))return p}return null},this.abort=function(){return this.abortController.abort(),this._abortController=null,this}}get abortController(){return this._abortController||(this._abortController=new AbortController),this._abortController}},Hf=new Oa,ln=class{constructor(e){this.manager=e!==void 0?e:Hf,this.crossOrigin="anonymous",this.withCredentials=!1,this.path="",this.resourcePath="",this.requestHeader={},typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}load(){}loadAsync(e,t){let n=this;return new Promise(function(s,r){n.load(e,s,t,r)})}parse(){}setCrossOrigin(e){return this.crossOrigin=e,this}setWithCredentials(e){return this.withCredentials=e,this}setPath(e){return this.path=e,this}setResourcePath(e){return this.resourcePath=e,this}setRequestHeader(e){return this.requestHeader=e,this}abort(){return this}};ln.DEFAULT_MATERIAL_NAME="__DEFAULT";var pi={},Al=class extends Error{constructor(e,t){super(e),this.response=t}},zn=class extends ln{constructor(e){super(e),this.mimeType="",this.responseType="",this._abortController=new AbortController}load(e,t,n,s){e===void 0&&(e=""),this.path!==void 0&&(e=this.path+e),e=this.manager.resolveURL(e);let r=jn.get(`file:${e}`);if(r!==void 0){this.manager.itemStart(e),setTimeout(()=>{t&&t(r),this.manager.itemEnd(e)},0);return}if(pi[e]!==void 0){pi[e].push({onLoad:t,onProgress:n,onError:s});return}pi[e]=[],pi[e].push({onLoad:t,onProgress:n,onError:s});let o=new Request(e,{headers:new Headers(this.requestHeader),credentials:this.withCredentials?"include":"same-origin",signal:typeof AbortSignal.any=="function"?AbortSignal.any([this._abortController.signal,this.manager.abortController.signal]):this._abortController.signal}),a=this.mimeType,c=this.responseType;fetch(o).then(l=>{if(l.status===200||l.status===0){if(l.status===0&&ke("FileLoader: HTTP Status 0 received."),typeof ReadableStream>"u"||l.body===void 0||l.body.getReader===void 0)return l;let h=pi[e],u=l.body.getReader(),f=l.headers.get("X-File-Size")||l.headers.get("Content-Length"),d=f?parseInt(f):0,p=d!==0,y=0,m=new ReadableStream({start(g){E();function E(){u.read().then(({done:S,value:x})=>{if(S)g.close();else{y+=x.byteLength;let A=new ProgressEvent("progress",{lengthComputable:p,loaded:y,total:d});for(let w=0,I=h.length;w<I;w++){let v=h[w];v.onProgress&&v.onProgress(A)}g.enqueue(x),E()}},S=>{g.error(S)})}}});return new Response(m)}else throw new Al(`fetch for "${l.url}" responded with ${l.status}: ${l.statusText}`,l)}).then(l=>{switch(c){case"arraybuffer":return l.arrayBuffer();case"blob":return l.blob();case"document":return l.text().then(h=>new DOMParser().parseFromString(h,a));case"json":return l.json();default:if(a==="")return l.text();{let u=/charset="?([^;"\s]*)"?/i.exec(a),f=u&&u[1]?u[1].toLowerCase():void 0,d=new TextDecoder(f);return l.arrayBuffer().then(p=>d.decode(p))}}}).then(l=>{jn.add(`file:${e}`,l);let h=pi[e];delete pi[e];for(let u=0,f=h.length;u<f;u++){let d=h[u];d.onLoad&&d.onLoad(l)}}).catch(l=>{let h=pi[e];if(h===void 0)throw this.manager.itemError(e),l;delete pi[e];for(let u=0,f=h.length;u<f;u++){let d=h[u];d.onError&&d.onError(l)}this.manager.itemError(e)}).finally(()=>{this.manager.itemEnd(e)}),this.manager.itemStart(e)}setResponseType(e){return this.responseType=e,this}setMimeType(e){return this.mimeType=e,this}abort(){return this._abortController.abort(),this._abortController=new AbortController,this}};var Ws=new WeakMap,Fa=class extends ln{constructor(e){super(e)}load(e,t,n,s){this.path!==void 0&&(e=this.path+e),e=this.manager.resolveURL(e);let r=this,o=jn.get(`image:${e}`);if(o!==void 0){if(o.complete===!0)r.manager.itemStart(e),setTimeout(function(){t&&t(o),r.manager.itemEnd(e)},0);else{let u=Ws.get(o);u===void 0&&(u=[],Ws.set(o,u)),u.push({onLoad:t,onError:s})}return o}let a=Ks("img");function c(){h(),t&&t(this);let u=Ws.get(this)||[];for(let f=0;f<u.length;f++){let d=u[f];d.onLoad&&d.onLoad(this)}Ws.delete(this),r.manager.itemEnd(e)}function l(u){h(),s&&s(u),jn.remove(`image:${e}`);let f=Ws.get(this)||[];for(let d=0;d<f.length;d++){let p=f[d];p.onError&&p.onError(u)}Ws.delete(this),r.manager.itemError(e),r.manager.itemEnd(e)}function h(){a.removeEventListener("load",c,!1),a.removeEventListener("error",l,!1)}return a.addEventListener("load",c,!1),a.addEventListener("error",l,!1),e.slice(0,5)!=="data:"&&this.crossOrigin!==void 0&&(a.crossOrigin=this.crossOrigin),jn.add(`image:${e}`,a),r.manager.itemStart(e),a.src=e,a}};var vs=class extends ln{constructor(e){super(e)}load(e,t,n,s){let r=new Tt,o=new Fa(this.manager);return o.setCrossOrigin(this.crossOrigin),o.setPath(this.path),o.load(e,function(a){r.image=a,r.needsUpdate=!0,t!==void 0&&t(r)},n,s),r}},rr=class extends At{constructor(e,t=1){super(),this.isLight=!0,this.type="Light",this.color=new Ge(e),this.intensity=t}dispose(){this.dispatchEvent({type:"dispose"})}copy(e,t){return super.copy(e,t),this.color.copy(e.color),this.intensity=e.intensity,this}toJSON(e){let t=super.toJSON(e);return t.object.color=this.color.getHex(),t.object.intensity=this.intensity,t}};var dl=new Je,qu=new B,Yu=new B,lo=class{constructor(e){this.camera=e,this.intensity=1,this.bias=0,this.biasNode=null,this.normalBias=0,this.radius=1,this.blurSamples=8,this.mapSize=new de(512,512),this.mapType=hn,this.map=null,this.mapPass=null,this.matrix=new Je,this.autoUpdate=!0,this.needsUpdate=!1,this._frustum=new er,this._frameExtents=new de(1,1),this._viewportCount=1,this._viewports=[new ft(0,0,1,1)]}getViewportCount(){return this._viewportCount}getFrustum(){return this._frustum}updateMatrices(e){let t=this.camera,n=this.matrix;qu.setFromMatrixPosition(e.matrixWorld),t.position.copy(qu),Yu.setFromMatrixPosition(e.target.matrixWorld),t.lookAt(Yu),t.updateMatrixWorld(),dl.multiplyMatrices(t.projectionMatrix,t.matrixWorldInverse),this._frustum.setFromProjectionMatrix(dl,t.coordinateSystem,t.reversedDepth),t.coordinateSystem===Zs||t.reversedDepth?n.set(.5,0,0,.5,0,.5,0,.5,0,0,1,0,0,0,0,1):n.set(.5,0,0,.5,0,.5,0,.5,0,0,.5,.5,0,0,0,1),n.multiply(dl)}getViewport(e){return this._viewports[e]}getFrameExtents(){return this._frameExtents}dispose(){this.map&&this.map.dispose(),this.mapPass&&this.mapPass.dispose()}copy(e){return this.camera=e.camera.clone(),this.intensity=e.intensity,this.bias=e.bias,this.radius=e.radius,this.autoUpdate=e.autoUpdate,this.needsUpdate=e.needsUpdate,this.normalBias=e.normalBias,this.blurSamples=e.blurSamples,this.mapSize.copy(e.mapSize),this.biasNode=e.biasNode,this}clone(){return new this.constructor().copy(this)}toJSON(){let e={};return this.intensity!==1&&(e.intensity=this.intensity),this.bias!==0&&(e.bias=this.bias),this.normalBias!==0&&(e.normalBias=this.normalBias),this.radius!==1&&(e.radius=this.radius),(this.mapSize.x!==512||this.mapSize.y!==512)&&(e.mapSize=this.mapSize.toArray()),e.camera=this.camera.toJSON(!1).object,delete e.camera.matrix,e}},ia=new B,sa=new Kt,Zn=new B,ho=class extends At{constructor(){super(),this.isCamera=!0,this.type="Camera",this.matrixWorldInverse=new Je,this.projectionMatrix=new Je,this.projectionMatrixInverse=new Je,this.coordinateSystem=Un,this._reversedDepth=!1}get reversedDepth(){return this._reversedDepth}copy(e,t){return super.copy(e,t),this.matrixWorldInverse.copy(e.matrixWorldInverse),this.projectionMatrix.copy(e.projectionMatrix),this.projectionMatrixInverse.copy(e.projectionMatrixInverse),this.coordinateSystem=e.coordinateSystem,this}getWorldDirection(e){return super.getWorldDirection(e).negate()}updateMatrixWorld(e){super.updateMatrixWorld(e),this.matrixWorld.decompose(ia,sa,Zn),Zn.x===1&&Zn.y===1&&Zn.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(ia,sa,Zn.set(1,1,1)).invert()}updateWorldMatrix(e,t,n=!1){super.updateWorldMatrix(e,t,n),this.matrixWorld.decompose(ia,sa,Zn),Zn.x===1&&Zn.y===1&&Zn.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(ia,sa,Zn.set(1,1,1)).invert()}clone(){return new this.constructor().copy(this)}},Fi=new B,Zu=new de,Ku=new de,Ct=class extends ho{constructor(e=50,t=1,n=.1,s=2e3){super(),this.isPerspectiveCamera=!0,this.type="PerspectiveCamera",this.fov=e,this.zoom=1,this.near=n,this.far=s,this.focus=10,this.aspect=t,this.view=null,this.filmGauge=35,this.filmOffset=0,this.updateProjectionMatrix()}copy(e,t){return super.copy(e,t),this.fov=e.fov,this.zoom=e.zoom,this.near=e.near,this.far=e.far,this.focus=e.focus,this.aspect=e.aspect,this.view=e.view===null?null:Object.assign({},e.view),this.filmGauge=e.filmGauge,this.filmOffset=e.filmOffset,this}setFocalLength(e){let t=.5*this.getFilmHeight()/e;this.fov=ps*2*Math.atan(t),this.updateProjectionMatrix()}getFocalLength(){let e=Math.tan(Or*.5*this.fov);return .5*this.getFilmHeight()/e}getEffectiveFOV(){return ps*2*Math.atan(Math.tan(Or*.5*this.fov)/this.zoom)}getFilmWidth(){return this.filmGauge*Math.min(this.aspect,1)}getFilmHeight(){return this.filmGauge/Math.max(this.aspect,1)}getViewBounds(e,t,n){Fi.set(-1,-1,.5).applyMatrix4(this.projectionMatrixInverse),t.set(Fi.x,Fi.y).multiplyScalar(-e/Fi.z),Fi.set(1,1,.5).applyMatrix4(this.projectionMatrixInverse),n.set(Fi.x,Fi.y).multiplyScalar(-e/Fi.z)}getViewSize(e,t){return this.getViewBounds(e,Zu,Ku),t.subVectors(Ku,Zu)}setViewOffset(e,t,n,s,r,o){this.aspect=e/t,this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=t,this.view.offsetX=n,this.view.offsetY=s,this.view.width=r,this.view.height=o,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){let e=this.near,t=e*Math.tan(Or*.5*this.fov)/this.zoom,n=2*t,s=this.aspect*n,r=-.5*s,o=this.view;if(this.view!==null&&this.view.enabled){let c=o.fullWidth,l=o.fullHeight;r+=o.offsetX*s/c,t-=o.offsetY*n/l,s*=o.width/c,n*=o.height/l}let a=this.filmOffset;a!==0&&(r+=e*a/this.getFilmWidth()),this.projectionMatrix.makePerspective(r,r+s,t,t-n,e,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){let t=super.toJSON(e);return t.object.fov=this.fov,t.object.zoom=this.zoom,t.object.near=this.near,t.object.far=this.far,t.object.focus=this.focus,t.object.aspect=this.aspect,this.view!==null&&(t.object.view=Object.assign({},this.view)),t.object.filmGauge=this.filmGauge,t.object.filmOffset=this.filmOffset,t}},wl=class extends lo{constructor(){super(new Ct(50,1,.5,500)),this.isSpotLightShadow=!0,this.focus=1,this.aspect=1}updateMatrices(e){let t=this.camera,n=ps*2*e.angle*this.focus,s=this.mapSize.width/this.mapSize.height*this.aspect,r=e.distance||t.far;(n!==t.fov||s!==t.aspect||r!==t.far)&&(t.fov=n,t.aspect=s,t.far=r,t.updateProjectionMatrix()),super.updateMatrices(e)}copy(e){return super.copy(e),this.focus=e.focus,this}},Si=class extends rr{constructor(e,t,n=0,s=Math.PI/3,r=0,o=2){super(e,t),this.isSpotLight=!0,this.type="SpotLight",this.position.copy(At.DEFAULT_UP),this.updateMatrix(),this.target=new At,this.distance=n,this.angle=s,this.penumbra=r,this.decay=o,this.map=null,this.shadow=new wl}get power(){return this.intensity*Math.PI}set power(e){this.intensity=e/Math.PI}dispose(){super.dispose(),this.shadow.dispose()}copy(e,t){return super.copy(e,t),this.distance=e.distance,this.angle=e.angle,this.penumbra=e.penumbra,this.decay=e.decay,this.target=e.target.clone(),this.map=e.map,this.shadow=e.shadow.clone(),this}toJSON(e){let t=super.toJSON(e);return t.object.distance=this.distance,t.object.angle=this.angle,t.object.decay=this.decay,t.object.penumbra=this.penumbra,t.object.target=this.target.uuid,this.map&&this.map.isTexture&&(t.object.map=this.map.toJSON(e).uuid),t.object.shadow=this.shadow.toJSON(),t}},Rl=class extends lo{constructor(){super(new Ct(90,1,.5,500)),this.isPointLightShadow=!0}},Hn=class extends rr{constructor(e,t,n=0,s=2){super(e,t),this.isPointLight=!0,this.type="PointLight",this.distance=n,this.decay=s,this.shadow=new Rl}get power(){return this.intensity*4*Math.PI}set power(e){this.intensity=e/(4*Math.PI)}dispose(){super.dispose(),this.shadow.dispose()}copy(e,t){return super.copy(e,t),this.distance=e.distance,this.decay=e.decay,this.shadow=e.shadow.clone(),this}toJSON(e){let t=super.toJSON(e);return t.object.distance=this.distance,t.object.decay=this.decay,t.object.shadow=this.shadow.toJSON(),t}},ti=class extends ho{constructor(e=-1,t=1,n=1,s=-1,r=.1,o=2e3){super(),this.isOrthographicCamera=!0,this.type="OrthographicCamera",this.zoom=1,this.view=null,this.left=e,this.right=t,this.top=n,this.bottom=s,this.near=r,this.far=o,this.updateProjectionMatrix()}copy(e,t){return super.copy(e,t),this.left=e.left,this.right=e.right,this.top=e.top,this.bottom=e.bottom,this.near=e.near,this.far=e.far,this.zoom=e.zoom,this.view=e.view===null?null:Object.assign({},e.view),this}setViewOffset(e,t,n,s,r,o){this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=t,this.view.offsetX=n,this.view.offsetY=s,this.view.width=r,this.view.height=o,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){let e=(this.right-this.left)/(2*this.zoom),t=(this.top-this.bottom)/(2*this.zoom),n=(this.right+this.left)/2,s=(this.top+this.bottom)/2,r=n-e,o=n+e,a=s+t,c=s-t;if(this.view!==null&&this.view.enabled){let l=(this.right-this.left)/this.view.fullWidth/this.zoom,h=(this.top-this.bottom)/this.view.fullHeight/this.zoom;r+=l*this.view.offsetX,o=r+l*this.view.width,a-=h*this.view.offsetY,c=a-h*this.view.height}this.projectionMatrix.makeOrthographic(r,o,a,c,this.near,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){let t=super.toJSON(e);return t.object.zoom=this.zoom,t.object.left=this.left,t.object.right=this.right,t.object.top=this.top,t.object.bottom=this.bottom,t.object.near=this.near,t.object.far=this.far,this.view!==null&&(t.object.view=Object.assign({},this.view)),t}},Cl=class extends lo{constructor(){super(new ti(-5,5,5,-5,.5,500)),this.isDirectionalLightShadow=!0}},uo=class extends rr{constructor(e,t){super(e,t),this.isDirectionalLight=!0,this.type="DirectionalLight",this.position.copy(At.DEFAULT_UP),this.updateMatrix(),this.target=new At,this.shadow=new Cl}dispose(){super.dispose(),this.shadow.dispose()}copy(e){return super.copy(e),this.target=e.target.clone(),this.shadow=e.shadow.clone(),this}toJSON(e){let t=super.toJSON(e);return t.object.shadow=this.shadow.toJSON(),t.object.target=this.target.uuid,t}};var _n=class{static extractUrlBase(e){let t=e.lastIndexOf("/");return t===-1?"./":e.slice(0,t+1)}static resolveURL(e,t){return typeof e!="string"||e===""?"":(/^https?:\/\//i.test(t)&&/^\//.test(e)&&(t=t.replace(/(^https?:\/\/[^\/]+).*/i,"$1")),/^(https?:)?\/\//i.test(e)||/^data:.*,.*$/i.test(e)||/^blob:.*$/i.test(e)?e:t+e)}};var pl=new WeakMap,fo=class extends ln{constructor(e){super(e),this.isImageBitmapLoader=!0,typeof createImageBitmap>"u"&&ke("ImageBitmapLoader: createImageBitmap() not supported."),typeof fetch>"u"&&ke("ImageBitmapLoader: fetch() not supported."),this.options={premultiplyAlpha:"none"},this._abortController=new AbortController}setOptions(e){return this.options=e,this}load(e,t,n,s){e===void 0&&(e=""),this.path!==void 0&&(e=this.path+e),e=this.manager.resolveURL(e);let r=this,o=jn.get(`image-bitmap:${e}`);if(o!==void 0){if(r.manager.itemStart(e),o.then){o.then(l=>{pl.has(o)===!0?(s&&s(pl.get(o)),r.manager.itemError(e),r.manager.itemEnd(e)):(t&&t(l),r.manager.itemEnd(e))});return}setTimeout(function(){t&&t(o),r.manager.itemEnd(e)},0);return}let a={};a.credentials=this.crossOrigin==="anonymous"?"same-origin":"include",a.headers=this.requestHeader,a.signal=typeof AbortSignal.any=="function"?AbortSignal.any([this._abortController.signal,this.manager.abortController.signal]):this._abortController.signal;let c=fetch(e,a).then(function(l){return l.blob()}).then(function(l){return createImageBitmap(l,Object.assign(r.options,{colorSpaceConversion:"none"}))}).then(function(l){jn.add(`image-bitmap:${e}`,l),t&&t(l),r.manager.itemEnd(e)}).catch(function(l){s&&s(l),pl.set(c,l),jn.remove(`image-bitmap:${e}`),r.manager.itemError(e),r.manager.itemEnd(e)});jn.add(`image-bitmap:${e}`,c),r.manager.itemStart(e)}abort(){return this._abortController.abort(),this._abortController=new AbortController,this}};var Xs=-90,qs=1,Ba=class extends At{constructor(e,t,n){super(),this.type="CubeCamera",this.renderTarget=n,this.coordinateSystem=null,this.activeMipmapLevel=0;let s=new Ct(Xs,qs,e,t);s.layers=this.layers,this.add(s);let r=new Ct(Xs,qs,e,t);r.layers=this.layers,this.add(r);let o=new Ct(Xs,qs,e,t);o.layers=this.layers,this.add(o);let a=new Ct(Xs,qs,e,t);a.layers=this.layers,this.add(a);let c=new Ct(Xs,qs,e,t);c.layers=this.layers,this.add(c);let l=new Ct(Xs,qs,e,t);l.layers=this.layers,this.add(l)}updateCoordinateSystem(){let e=this.coordinateSystem,t=this.children.concat(),[n,s,r,o,a,c]=t;for(let l of t)this.remove(l);if(e===Un)n.up.set(0,1,0),n.lookAt(1,0,0),s.up.set(0,1,0),s.lookAt(-1,0,0),r.up.set(0,0,-1),r.lookAt(0,1,0),o.up.set(0,0,1),o.lookAt(0,-1,0),a.up.set(0,1,0),a.lookAt(0,0,1),c.up.set(0,1,0),c.lookAt(0,0,-1);else if(e===Zs)n.up.set(0,-1,0),n.lookAt(-1,0,0),s.up.set(0,-1,0),s.lookAt(1,0,0),r.up.set(0,0,1),r.lookAt(0,1,0),o.up.set(0,0,-1),o.lookAt(0,-1,0),a.up.set(0,-1,0),a.lookAt(0,0,1),c.up.set(0,-1,0),c.lookAt(0,0,-1);else throw new Error("THREE.CubeCamera.updateCoordinateSystem(): Invalid coordinate system: "+e);for(let l of t)this.add(l),l.updateMatrixWorld()}update(e,t){this.parent===null&&this.updateMatrixWorld();let{renderTarget:n,activeMipmapLevel:s}=this;this.coordinateSystem!==e.coordinateSystem&&(this.coordinateSystem=e.coordinateSystem,this.updateCoordinateSystem());let[r,o,a,c,l,h]=this.children,u=e.getRenderTarget(),f=e.getActiveCubeFace(),d=e.getActiveMipmapLevel(),p=e.xr.enabled;e.xr.enabled=!1;let y=n.texture.generateMipmaps;n.texture.generateMipmaps=!1;let m=!1;e.isWebGLRenderer===!0?m=e.state.buffers.depth.getReversed():m=e.reversedDepthBuffer,e.setRenderTarget(n,0,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,r),e.setRenderTarget(n,1,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,o),e.setRenderTarget(n,2,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,a),e.setRenderTarget(n,3,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,c),e.setRenderTarget(n,4,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,l),n.texture.generateMipmaps=y,e.setRenderTarget(n,5,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,h),e.setRenderTarget(u,f,d),e.xr.enabled=p,n.texture.needsPMREMUpdate=!0}},ka=class extends Ct{constructor(e=[]){super(),this.isArrayCamera=!0,this.isMultiViewCamera=!1,this.cameras=e}};var eh="\\[\\]\\.:\\/",ym=new RegExp("["+eh+"]","g"),th="[^"+eh+"]",vm="[^"+eh.replace("\\.","")+"]",bm=/((?:WC+[\/:])*)/.source.replace("WC",th),Mm=/(WCOD+)?/.source.replace("WCOD",vm),Sm=/(?:\.(WC+)(?:\[(.+)\])?)?/.source.replace("WC",th),Tm=/\.(WC+)(?:\[(.+)\])?/.source.replace("WC",th),Em=new RegExp("^"+bm+Mm+Sm+Tm+"$"),Am=["material","materials","bones","map"],Pl=class{constructor(e,t,n){let s=n||xt.parseTrackName(t);this._targetGroup=e,this._bindings=e.subscribe_(t,s)}getValue(e,t){this.bind();let n=this._targetGroup.nCachedObjects_,s=this._bindings[n];s!==void 0&&s.getValue(e,t)}setValue(e,t){let n=this._bindings;for(let s=this._targetGroup.nCachedObjects_,r=n.length;s!==r;++s)n[s].setValue(e,t)}bind(){let e=this._bindings;for(let t=this._targetGroup.nCachedObjects_,n=e.length;t!==n;++t)e[t].bind()}unbind(){let e=this._bindings;for(let t=this._targetGroup.nCachedObjects_,n=e.length;t!==n;++t)e[t].unbind()}},xt=class i{constructor(e,t,n){this.path=t,this.parsedPath=n||i.parseTrackName(t),this.node=i.findNode(e,this.parsedPath.nodeName),this.rootNode=e,this.getValue=this._getValue_unbound,this.setValue=this._setValue_unbound}static create(e,t,n){return e&&e.isAnimationObjectGroup?new i.Composite(e,t,n):new i(e,t,n)}static sanitizeNodeName(e){return e.replace(/\s/g,"_").replace(ym,"")}static parseTrackName(e){let t=Em.exec(e);if(t===null)throw new Error("THREE.PropertyBinding: Cannot parse trackName: "+e);let n={nodeName:t[2],objectName:t[3],objectIndex:t[4],propertyName:t[5],propertyIndex:t[6]},s=n.nodeName&&n.nodeName.lastIndexOf(".");if(s!==void 0&&s!==-1){let r=n.nodeName.substring(s+1);Am.indexOf(r)!==-1&&(n.nodeName=n.nodeName.substring(0,s),n.objectName=r)}if(n.propertyName===null||n.propertyName.length===0)throw new Error("THREE.PropertyBinding: can not parse propertyName from trackName: "+e);return n}static findNode(e,t){if(t===void 0||t===""||t==="."||t===-1||t===e.name||t===e.uuid)return e;if(e.skeleton){let n=e.skeleton.getBoneByName(t);if(n!==void 0)return n}if(e.children){let n=function(r){for(let o=0;o<r.length;o++){let a=r[o];if(a.name===t||a.uuid===t)return a;let c=n(a.children);if(c)return c}return null},s=n(e.children);if(s)return s}return null}_getValue_unavailable(){}_setValue_unavailable(){}_getValue_direct(e,t){e[t]=this.targetObject[this.propertyName]}_getValue_array(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)e[t++]=n[s]}_getValue_arrayElement(e,t){e[t]=this.resolvedProperty[this.propertyIndex]}_getValue_toArray(e,t){this.resolvedProperty.toArray(e,t)}_setValue_direct(e,t){this.targetObject[this.propertyName]=e[t]}_setValue_direct_setNeedsUpdate(e,t){this.targetObject[this.propertyName]=e[t],this.targetObject.needsUpdate=!0}_setValue_direct_setMatrixWorldNeedsUpdate(e,t){this.targetObject[this.propertyName]=e[t],this.targetObject.matrixWorldNeedsUpdate=!0}_setValue_array(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)n[s]=e[t++]}_setValue_array_setNeedsUpdate(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)n[s]=e[t++];this.targetObject.needsUpdate=!0}_setValue_array_setMatrixWorldNeedsUpdate(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)n[s]=e[t++];this.targetObject.matrixWorldNeedsUpdate=!0}_setValue_arrayElement(e,t){this.resolvedProperty[this.propertyIndex]=e[t]}_setValue_arrayElement_setNeedsUpdate(e,t){this.resolvedProperty[this.propertyIndex]=e[t],this.targetObject.needsUpdate=!0}_setValue_arrayElement_setMatrixWorldNeedsUpdate(e,t){this.resolvedProperty[this.propertyIndex]=e[t],this.targetObject.matrixWorldNeedsUpdate=!0}_setValue_fromArray(e,t){this.resolvedProperty.fromArray(e,t)}_setValue_fromArray_setNeedsUpdate(e,t){this.resolvedProperty.fromArray(e,t),this.targetObject.needsUpdate=!0}_setValue_fromArray_setMatrixWorldNeedsUpdate(e,t){this.resolvedProperty.fromArray(e,t),this.targetObject.matrixWorldNeedsUpdate=!0}_getValue_unbound(e,t){this.bind(),this.getValue(e,t)}_setValue_unbound(e,t){this.bind(),this.setValue(e,t)}bind(){let e=this.node,t=this.parsedPath,n=t.objectName,s=t.propertyName,r=t.propertyIndex;if(e||(e=i.findNode(this.rootNode,t.nodeName),this.node=e),this.getValue=this._getValue_unavailable,this.setValue=this._setValue_unavailable,!e){ke("PropertyBinding: No target node found for track: "+this.path+".");return}if(n){let l=t.objectIndex;switch(n){case"materials":if(!e.material){Ke("PropertyBinding: Can not bind to material as node does not have a material.",this);return}if(!e.material.materials){Ke("PropertyBinding: Can not bind to material.materials as node.material does not have a materials array.",this);return}e=e.material.materials;break;case"bones":if(!e.skeleton){Ke("PropertyBinding: Can not bind to bones as node does not have a skeleton.",this);return}e=e.skeleton.bones;for(let h=0;h<e.length;h++)if(e[h].name===l){l=h;break}break;case"map":if("map"in e){e=e.map;break}if(!e.material){Ke("PropertyBinding: Can not bind to material as node does not have a material.",this);return}if(!e.material.map){Ke("PropertyBinding: Can not bind to material.map as node.material does not have a map.",this);return}e=e.material.map;break;default:if(e[n]===void 0){Ke("PropertyBinding: Can not bind to objectName of node undefined.",this);return}e=e[n]}if(l!==void 0){if(e[l]===void 0){Ke("PropertyBinding: Trying to bind to objectIndex of objectName, but is undefined.",this,e);return}e=e[l]}}let o=e[s];if(o===void 0){let l=t.nodeName;Ke("PropertyBinding: Trying to update property for track: "+l+"."+s+" but it wasn't found.",e);return}let a=this.Versioning.None;this.targetObject=e,e.isMaterial===!0?a=this.Versioning.NeedsUpdate:e.isObject3D===!0&&(a=this.Versioning.MatrixWorldNeedsUpdate);let c=this.BindingType.Direct;if(r!==void 0){if(s==="morphTargetInfluences"){if(!e.geometry){Ke("PropertyBinding: Can not bind to morphTargetInfluences because node does not have a geometry.",this);return}if(!e.geometry.morphAttributes){Ke("PropertyBinding: Can not bind to morphTargetInfluences because node does not have a geometry.morphAttributes.",this);return}e.morphTargetDictionary[r]!==void 0&&(r=e.morphTargetDictionary[r])}c=this.BindingType.ArrayElement,this.resolvedProperty=o,this.propertyIndex=r}else o.fromArray!==void 0&&o.toArray!==void 0?(c=this.BindingType.HasFromToArray,this.resolvedProperty=o):Array.isArray(o)?(c=this.BindingType.EntireArray,this.resolvedProperty=o):this.propertyName=s;this.getValue=this.GetterByBindingType[c],this.setValue=this.SetterByBindingTypeAndVersioning[c][a]}unbind(){this.node=null,this.getValue=this._getValue_unbound,this.setValue=this._setValue_unbound}};xt.Composite=Pl;xt.prototype.BindingType={Direct:0,EntireArray:1,ArrayElement:2,HasFromToArray:3};xt.prototype.Versioning={None:0,NeedsUpdate:1,MatrixWorldNeedsUpdate:2};xt.prototype.GetterByBindingType=[xt.prototype._getValue_direct,xt.prototype._getValue_array,xt.prototype._getValue_arrayElement,xt.prototype._getValue_toArray];xt.prototype.SetterByBindingTypeAndVersioning=[[xt.prototype._setValue_direct,xt.prototype._setValue_direct_setNeedsUpdate,xt.prototype._setValue_direct_setMatrixWorldNeedsUpdate],[xt.prototype._setValue_array,xt.prototype._setValue_array_setNeedsUpdate,xt.prototype._setValue_array_setMatrixWorldNeedsUpdate],[xt.prototype._setValue_arrayElement,xt.prototype._setValue_arrayElement_setNeedsUpdate,xt.prototype._setValue_arrayElement_setMatrixWorldNeedsUpdate],[xt.prototype._setValue_fromArray,xt.prototype._setValue_fromArray_setNeedsUpdate,xt.prototype._setValue_fromArray_setMatrixWorldNeedsUpdate]];var Wv=new Float32Array(1);var or=class{constructor(e=1,t=0,n=0){this.radius=e,this.phi=t,this.theta=n}set(e,t,n){return this.radius=e,this.phi=t,this.theta=n,this}copy(e){return this.radius=e.radius,this.phi=e.phi,this.theta=e.theta,this}makeSafe(){return this.phi=et(this.phi,1e-6,Math.PI-1e-6),this}setFromVector3(e){return this.setFromCartesianCoords(e.x,e.y,e.z)}setFromCartesianCoords(e,t,n){return this.radius=Math.sqrt(e*e+t*t+n*n),this.radius===0?(this.theta=0,this.phi=0):(this.theta=Math.atan2(e,n),this.phi=Math.acos(et(t/this.radius,-1,1))),this}clone(){return new this.constructor().copy(this)}};var ah=class ah{constructor(e,t,n,s){this.elements=[1,0,0,1],e!==void 0&&this.set(e,t,n,s)}identity(){return this.set(1,0,0,1),this}fromArray(e,t=0){for(let n=0;n<4;n++)this.elements[n]=e[n+t];return this}set(e,t,n,s){let r=this.elements;return r[0]=e,r[2]=t,r[1]=n,r[3]=s,this}};ah.prototype.isMatrix2=!0;var Il=ah,ju=new de,qi=class{constructor(e=new de(1/0,1/0),t=new de(-1/0,-1/0)){this.isBox2=!0,this.min=e,this.max=t}set(e,t){return this.min.copy(e),this.max.copy(t),this}setFromPoints(e){this.makeEmpty();for(let t=0,n=e.length;t<n;t++)this.expandByPoint(e[t]);return this}setFromCenterAndSize(e,t){let n=ju.copy(t).multiplyScalar(.5);return this.min.copy(e).sub(n),this.max.copy(e).add(n),this}clone(){return new this.constructor().copy(this)}copy(e){return this.min.copy(e.min),this.max.copy(e.max),this}makeEmpty(){return this.min.x=this.min.y=1/0,this.max.x=this.max.y=-1/0,this}isEmpty(){return this.max.x<this.min.x||this.max.y<this.min.y}getCenter(e){return this.isEmpty()?e.set(0,0):e.addVectors(this.min,this.max).multiplyScalar(.5)}getSize(e){return this.isEmpty()?e.set(0,0):e.subVectors(this.max,this.min)}expandByPoint(e){return this.min.min(e),this.max.max(e),this}expandByVector(e){return this.min.sub(e),this.max.add(e),this}expandByScalar(e){return this.min.addScalar(-e),this.max.addScalar(e),this}containsPoint(e){return e.x>=this.min.x&&e.x<=this.max.x&&e.y>=this.min.y&&e.y<=this.max.y}containsBox(e){return this.min.x<=e.min.x&&e.max.x<=this.max.x&&this.min.y<=e.min.y&&e.max.y<=this.max.y}getParameter(e,t){return t.set((e.x-this.min.x)/(this.max.x-this.min.x),(e.y-this.min.y)/(this.max.y-this.min.y))}intersectsBox(e){return e.max.x>=this.min.x&&e.min.x<=this.max.x&&e.max.y>=this.min.y&&e.min.y<=this.max.y}clampPoint(e,t){return t.copy(e).clamp(this.min,this.max)}distanceToPoint(e){return this.clampPoint(e,ju).distanceTo(e)}intersect(e){return this.min.max(e.min),this.max.min(e.max),this.isEmpty()&&this.makeEmpty(),this}union(e){return this.min.min(e.min),this.max.max(e.max),this}translate(e){return this.min.add(e),this.max.add(e),this}equals(e){return e.min.equals(this.min)&&e.max.equals(this.max)}};var Vn=class{constructor(){this.type="ShapePath",this.color=new Ge,this.subPaths=[],this.currentPath=null,this.userData={}}moveTo(e,t){return this.currentPath=new tn,this.subPaths.push(this.currentPath),this.currentPath.moveTo(e,t),this}lineTo(e,t){return this.currentPath.lineTo(e,t),this}quadraticCurveTo(e,t,n,s){return this.currentPath.quadraticCurveTo(e,t,n,s),this}bezierCurveTo(e,t,n,s,r,o){return this.currentPath.bezierCurveTo(e,t,n,s,r,o),this}splineThru(e){return this.currentPath.splineThru(e),this}toShapes(){function e(c,l){let h=!1,u=l.length;for(let f=0,d=u-1;f<u;d=f++){let p=l[f],y=l[d];p.y>c.y!=y.y>c.y&&c.x<(y.x-p.x)*(c.y-p.y)/(y.y-p.y)+p.x&&(h=!h)}return h}function t(c,l){let h=l.getCenter(new de);if(e(h,c))return h;let u=h.y,f=[],d=c.length;for(let p=0;p<d;p++){let y=c[p],m=c[(p+1)%d];if(y.y>u!=m.y>u){let g=y.x+(u-y.y)*(m.x-y.x)/(m.y-y.y);f.push(g)}}return f.length>1&&(f.sort((p,y)=>p-y),h.x=(f[0]+f[1])/2),h}let n=this.userData.style&&this.userData.style.fillRule||"nonzero";n!=="nonzero"&&n!=="evenodd"&&(ke('Fill-rule "'+n+'" is not supported, falling back to "nonzero".'),n="nonzero");let s=n==="nonzero"?(c=>c!==0):(c=>(c&1)!==0),r=[];for(let c of this.subPaths){let l=c.getPoints();if(l.length<3)continue;let h=Kn.area(l);if(h===0)continue;let u=new qi;for(let f=0;f<l.length;f++)u.expandByPoint(l[f]);r.push({subPath:c,points:l,boundingBox:u,interiorPoint:t(l,u),absArea:Math.abs(h),winding:h<0?-1:1,container:null,exclude:!1,role:null})}r.sort((c,l)=>l.absArea-c.absArea);for(let c=0;c<r.length;c++){let l=r[c],h=0;for(let u=c-1;u>=0;u--){let f=r[u];if(f.boundingBox.containsBox(l.boundingBox)&&e(l.interiorPoint,f.points)){l.container=f.exclude?f.container:f,h=f.winding,l.winding+=h;break}}s(l.winding)===s(h)&&(l.exclude=!0)}for(let c of r)c.exclude||(c.role=c.container===null||c.container.role==="hole"?"outer":"hole");let o=[],a=new Map;for(let c of r){if(c.exclude||c.role!=="outer")continue;let l=new kn;l.curves=c.subPath.curves,o.push(l),a.set(c,l)}for(let c of r){if(c.exclude||c.role!=="hole")continue;let l=a.get(c.container);if(!l)continue;let h=new tn;h.curves=c.subPath.curves,l.holes.push(h)}return o}},po=class extends Fn{constructor(e,t=null){super(),this.object=e,this.domElement=t,this.enabled=!0,this.state=-1,this.keys={},this.mouseButtons={LEFT:null,MIDDLE:null,RIGHT:null},this.touches={ONE:null,TWO:null}}connect(e){if(e===void 0){ke("Controls: connect() now requires an element.");return}this.domElement!==null&&this.disconnect(),this.domElement=e}disconnect(){}dispose(){}update(){}};function nh(i,e,t,n){let s=wm(n);switch(t){case Zl:return i*e;case fr:return i*e/s.components*s.byteLength;case qa:return i*e/s.components*s.byteLength;case Ji:return i*e*2/s.components*s.byteLength;case Ya:return i*e*2/s.components*s.byteLength;case Kl:return i*e*3/s.components*s.byteLength;case vn:return i*e*4/s.components*s.byteLength;case Za:return i*e*4/s.components*s.byteLength;case _o:case xo:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*8;case yo:case vo:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case ja:case $a:return Math.max(i,16)*Math.max(e,8)/4;case Ka:case Ja:return Math.max(i,8)*Math.max(e,8)/2;case Qa:case ec:case nc:case ic:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*8;case tc:case bo:case sc:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case rc:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case oc:return Math.floor((i+4)/5)*Math.floor((e+3)/4)*16;case ac:return Math.floor((i+4)/5)*Math.floor((e+4)/5)*16;case cc:return Math.floor((i+5)/6)*Math.floor((e+4)/5)*16;case lc:return Math.floor((i+5)/6)*Math.floor((e+5)/6)*16;case hc:return Math.floor((i+7)/8)*Math.floor((e+4)/5)*16;case uc:return Math.floor((i+7)/8)*Math.floor((e+5)/6)*16;case fc:return Math.floor((i+7)/8)*Math.floor((e+7)/8)*16;case dc:return Math.floor((i+9)/10)*Math.floor((e+4)/5)*16;case pc:return Math.floor((i+9)/10)*Math.floor((e+5)/6)*16;case mc:return Math.floor((i+9)/10)*Math.floor((e+7)/8)*16;case gc:return Math.floor((i+9)/10)*Math.floor((e+9)/10)*16;case _c:return Math.floor((i+11)/12)*Math.floor((e+9)/10)*16;case xc:return Math.floor((i+11)/12)*Math.floor((e+11)/12)*16;case yc:case vc:case bc:return Math.ceil(i/4)*Math.ceil(e/4)*16;case Mc:case Sc:return Math.ceil(i/4)*Math.ceil(e/4)*8;case Mo:case Tc:return Math.ceil(i/4)*Math.ceil(e/4)*16}throw new Error(`Unable to determine texture byte length for ${t} format.`)}function wm(i){switch(i){case hn:case Wl:return{byteLength:1,components:1};case hr:case Xl:case ni:return{byteLength:2,components:1};case Wa:case Xa:return{byteLength:2,components:4};case Wn:case Ga:case nn:return{byteLength:4,components:1};case ql:case Yl:return{byteLength:4,components:3}}throw new Error(`THREE.TextureUtils: Unknown texture type ${i}.`)}typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("register",{detail:{revision:"185"}}));typeof window<"u"&&(window.__THREE__?ke("WARNING: Multiple instances of Three.js being imported."):window.__THREE__="185");function hd(){let i=null,e=!1,t=null,n=null;function s(r,o){t(r,o),n=i.requestAnimationFrame(s)}return{start:function(){e!==!0&&t!==null&&i!==null&&(n=i.requestAnimationFrame(s),e=!0)},stop:function(){i!==null&&i.cancelAnimationFrame(n),e=!1},setAnimationLoop:function(r){t=r},setContext:function(r){i=r}}}function Cm(i){let e=new WeakMap;function t(a,c){let l=a.array,h=a.usage,u=l.byteLength,f=i.createBuffer();i.bindBuffer(c,f),i.bufferData(c,l,h),a.onUploadCallback();let d;if(l instanceof Float32Array)d=i.FLOAT;else if(typeof Float16Array<"u"&&l instanceof Float16Array)d=i.HALF_FLOAT;else if(l instanceof Uint16Array)a.isFloat16BufferAttribute?d=i.HALF_FLOAT:d=i.UNSIGNED_SHORT;else if(l instanceof Int16Array)d=i.SHORT;else if(l instanceof Uint32Array)d=i.UNSIGNED_INT;else if(l instanceof Int32Array)d=i.INT;else if(l instanceof Int8Array)d=i.BYTE;else if(l instanceof Uint8Array)d=i.UNSIGNED_BYTE;else if(l instanceof Uint8ClampedArray)d=i.UNSIGNED_BYTE;else throw new Error("THREE.WebGLAttributes: Unsupported buffer data format: "+l);return{buffer:f,type:d,bytesPerElement:l.BYTES_PER_ELEMENT,version:a.version,size:u}}function n(a,c,l){let h=c.array,u=c.updateRanges;if(i.bindBuffer(l,a),u.length===0)i.bufferSubData(l,0,h);else{u.sort((d,p)=>d.start-p.start);let f=0;for(let d=1;d<u.length;d++){let p=u[f],y=u[d];y.start<=p.start+p.count+1?p.count=Math.max(p.count,y.start+y.count-p.start):(++f,u[f]=y)}u.length=f+1;for(let d=0,p=u.length;d<p;d++){let y=u[d];i.bufferSubData(l,y.start*h.BYTES_PER_ELEMENT,h,y.start,y.count)}c.clearUpdateRanges()}c.onUploadCallback()}function s(a){return a.isInterleavedBufferAttribute&&(a=a.data),e.get(a)}function r(a){a.isInterleavedBufferAttribute&&(a=a.data);let c=e.get(a);c&&(i.deleteBuffer(c.buffer),e.delete(a))}function o(a,c){if(a.isInterleavedBufferAttribute&&(a=a.data),a.isGLBufferAttribute){let h=e.get(a);(!h||h.version<a.version)&&e.set(a,{buffer:a.buffer,type:a.type,bytesPerElement:a.elementSize,version:a.version});return}let l=e.get(a);if(l===void 0)e.set(a,t(a,c));else if(l.version<a.version){if(l.size!==a.array.byteLength)throw new Error("THREE.WebGLAttributes: The size of the buffer attribute's array buffer does not match the original size. Resizing buffer attributes is not supported.");n(l.buffer,a,c),l.version=a.version}}return{get:s,remove:r,update:o}}var Pm=`#ifdef USE_ALPHAHASH
	if ( diffuseColor.a < getAlphaHashThreshold( vPosition ) ) discard;
#endif`,Im=`#ifdef USE_ALPHAHASH
	const float ALPHA_HASH_SCALE = 0.05;
	float hash2D( vec2 value ) {
		return fract( 1.0e4 * sin( 17.0 * value.x + 0.1 * value.y ) * ( 0.1 + abs( sin( 13.0 * value.y + value.x ) ) ) );
	}
	float hash3D( vec3 value ) {
		return hash2D( vec2( hash2D( value.xy ), value.z ) );
	}
	float getAlphaHashThreshold( vec3 position ) {
		float maxDeriv = max(
			length( dFdx( position.xyz ) ),
			length( dFdy( position.xyz ) )
		);
		float pixScale = 1.0 / ( ALPHA_HASH_SCALE * maxDeriv );
		vec2 pixScales = vec2(
			exp2( floor( log2( pixScale ) ) ),
			exp2( ceil( log2( pixScale ) ) )
		);
		vec2 alpha = vec2(
			hash3D( floor( pixScales.x * position.xyz ) ),
			hash3D( floor( pixScales.y * position.xyz ) )
		);
		float lerpFactor = fract( log2( pixScale ) );
		float x = ( 1.0 - lerpFactor ) * alpha.x + lerpFactor * alpha.y;
		float a = min( lerpFactor, 1.0 - lerpFactor );
		vec3 cases = vec3(
			x * x / ( 2.0 * a * ( 1.0 - a ) ),
			( x - 0.5 * a ) / ( 1.0 - a ),
			1.0 - ( ( 1.0 - x ) * ( 1.0 - x ) / ( 2.0 * a * ( 1.0 - a ) ) )
		);
		float threshold = ( x < ( 1.0 - a ) )
			? ( ( x < a ) ? cases.x : cases.y )
			: cases.z;
		return clamp( threshold , 1.0e-6, 1.0 );
	}
#endif`,Lm=`#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, vAlphaMapUv ).g;
#endif`,Dm=`#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,Nm=`#ifdef USE_ALPHATEST
	#ifdef ALPHA_TO_COVERAGE
	diffuseColor.a = smoothstep( alphaTest, alphaTest + fwidth( diffuseColor.a ), diffuseColor.a );
	if ( diffuseColor.a == 0.0 ) discard;
	#else
	if ( diffuseColor.a < alphaTest ) discard;
	#endif
#endif`,Um=`#ifdef USE_ALPHATEST
	uniform float alphaTest;
#endif`,Om=`#ifdef USE_AOMAP
	float ambientOcclusion = ( texture2D( aoMap, vAoMapUv ).r - 1.0 ) * aoMapIntensity + 1.0;
	reflectedLight.indirectDiffuse *= ambientOcclusion;
	#if defined( USE_CLEARCOAT ) 
		clearcoatSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_SHEEN ) 
		sheenSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_ENVMAP ) && defined( STANDARD )
		float dotNV = saturate( dot( geometryNormal, geometryViewDir ) );
		reflectedLight.indirectSpecular *= computeSpecularOcclusion( dotNV, ambientOcclusion, material.roughness );
	#endif
#endif`,Fm=`#ifdef USE_AOMAP
	uniform sampler2D aoMap;
	uniform float aoMapIntensity;
#endif`,Bm=`#ifdef USE_BATCHING
	#if ! defined( GL_ANGLE_multi_draw )
	#define gl_DrawID _gl_DrawID
	uniform int _gl_DrawID;
	#endif
	uniform highp sampler2D batchingTexture;
	uniform highp usampler2D batchingIdTexture;
	mat4 getBatchingMatrix( const in float i ) {
		int size = textureSize( batchingTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( batchingTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( batchingTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( batchingTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( batchingTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
	float getIndirectIndex( const in int i ) {
		int size = textureSize( batchingIdTexture, 0 ).x;
		int x = i % size;
		int y = i / size;
		return float( texelFetch( batchingIdTexture, ivec2( x, y ), 0 ).r );
	}
#endif
#ifdef USE_BATCHING_COLOR
	uniform sampler2D batchingColorTexture;
	vec4 getBatchingColor( const in float i ) {
		int size = textureSize( batchingColorTexture, 0 ).x;
		int j = int( i );
		int x = j % size;
		int y = j / size;
		return texelFetch( batchingColorTexture, ivec2( x, y ), 0 );
	}
#endif`,km=`#ifdef USE_BATCHING
	mat4 batchingMatrix = getBatchingMatrix( getIndirectIndex( gl_DrawID ) );
#endif`,zm=`vec3 transformed = vec3( position );
#ifdef USE_ALPHAHASH
	vPosition = vec3( position );
#endif`,Hm=`vec3 objectNormal = vec3( normal );
#ifdef USE_TANGENT
	vec3 objectTangent = vec3( tangent.xyz );
#endif`,Vm=`float G_BlinnPhong_Implicit( ) {
	return 0.25;
}
float D_BlinnPhong( const in float shininess, const in float dotNH ) {
	return RECIPROCAL_PI * ( shininess * 0.5 + 1.0 ) * pow( dotNH, shininess );
}
vec3 BRDF_BlinnPhong( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in vec3 specularColor, const in float shininess ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( specularColor, 1.0, dotVH );
	float G = G_BlinnPhong_Implicit( );
	float D = D_BlinnPhong( shininess, dotNH );
	return F * ( G * D );
} // validated`,Gm=`#ifdef USE_IRIDESCENCE
	const mat3 XYZ_TO_REC709 = mat3(
		 3.2404542, -0.9692660,  0.0556434,
		-1.5371385,  1.8760108, -0.2040259,
		-0.4985314,  0.0415560,  1.0572252
	);
	vec3 Fresnel0ToIor( vec3 fresnel0 ) {
		vec3 sqrtF0 = sqrt( fresnel0 );
		return ( vec3( 1.0 ) + sqrtF0 ) / ( vec3( 1.0 ) - sqrtF0 );
	}
	vec3 IorToFresnel0( vec3 transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - vec3( incidentIor ) ) / ( transmittedIor + vec3( incidentIor ) ) );
	}
	float IorToFresnel0( float transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - incidentIor ) / ( transmittedIor + incidentIor ));
	}
	vec3 evalSensitivity( float OPD, vec3 shift ) {
		float phase = 2.0 * PI * OPD * 1.0e-9;
		vec3 val = vec3( 5.4856e-13, 4.4201e-13, 5.2481e-13 );
		vec3 pos = vec3( 1.6810e+06, 1.7953e+06, 2.2084e+06 );
		vec3 var = vec3( 4.3278e+09, 9.3046e+09, 6.6121e+09 );
		vec3 xyz = val * sqrt( 2.0 * PI * var ) * cos( pos * phase + shift ) * exp( - pow2( phase ) * var );
		xyz.x += 9.7470e-14 * sqrt( 2.0 * PI * 4.5282e+09 ) * cos( 2.2399e+06 * phase + shift[ 0 ] ) * exp( - 4.5282e+09 * pow2( phase ) );
		xyz /= 1.0685e-7;
		vec3 rgb = XYZ_TO_REC709 * xyz;
		return rgb;
	}
	vec3 evalIridescence( float outsideIOR, float eta2, float cosTheta1, float thinFilmThickness, vec3 baseF0 ) {
		vec3 I;
		float iridescenceIOR = mix( outsideIOR, eta2, smoothstep( 0.0, 0.03, thinFilmThickness ) );
		float sinTheta2Sq = pow2( outsideIOR / iridescenceIOR ) * ( 1.0 - pow2( cosTheta1 ) );
		float cosTheta2Sq = 1.0 - sinTheta2Sq;
		if ( cosTheta2Sq < 0.0 ) {
			return vec3( 1.0 );
		}
		float cosTheta2 = sqrt( cosTheta2Sq );
		float R0 = IorToFresnel0( iridescenceIOR, outsideIOR );
		float R12 = F_Schlick( R0, 1.0, cosTheta1 );
		float T121 = 1.0 - R12;
		float phi12 = 0.0;
		if ( iridescenceIOR < outsideIOR ) phi12 = PI;
		float phi21 = PI - phi12;
		vec3 baseIOR = Fresnel0ToIor( clamp( baseF0, 0.0, 0.9999 ) );		vec3 R1 = IorToFresnel0( baseIOR, iridescenceIOR );
		vec3 R23 = F_Schlick( R1, 1.0, cosTheta2 );
		vec3 phi23 = vec3( 0.0 );
		if ( baseIOR[ 0 ] < iridescenceIOR ) phi23[ 0 ] = PI;
		if ( baseIOR[ 1 ] < iridescenceIOR ) phi23[ 1 ] = PI;
		if ( baseIOR[ 2 ] < iridescenceIOR ) phi23[ 2 ] = PI;
		float OPD = 2.0 * iridescenceIOR * thinFilmThickness * cosTheta2;
		vec3 phi = vec3( phi21 ) + phi23;
		vec3 R123 = clamp( R12 * R23, 1e-5, 0.9999 );
		vec3 r123 = sqrt( R123 );
		vec3 Rs = pow2( T121 ) * R23 / ( vec3( 1.0 ) - R123 );
		vec3 C0 = R12 + Rs;
		I = C0;
		vec3 Cm = Rs - T121;
		for ( int m = 1; m <= 2; ++ m ) {
			Cm *= r123;
			vec3 Sm = 2.0 * evalSensitivity( float( m ) * OPD, float( m ) * phi );
			I += Cm * Sm;
		}
		return max( I, vec3( 0.0 ) );
	}
#endif`,Wm=`#ifdef USE_BUMPMAP
	uniform sampler2D bumpMap;
	uniform float bumpScale;
	vec2 dHdxy_fwd() {
		vec2 dSTdx = dFdx( vBumpMapUv );
		vec2 dSTdy = dFdy( vBumpMapUv );
		float Hll = bumpScale * texture2D( bumpMap, vBumpMapUv ).x;
		float dBx = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdx ).x - Hll;
		float dBy = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdy ).x - Hll;
		return vec2( dBx, dBy );
	}
	vec3 perturbNormalArb( vec3 surf_pos, vec3 surf_norm, vec2 dHdxy, float faceDirection ) {
		vec3 vSigmaX = normalize( dFdx( surf_pos.xyz ) );
		vec3 vSigmaY = normalize( dFdy( surf_pos.xyz ) );
		vec3 vN = surf_norm;
		vec3 R1 = cross( vSigmaY, vN );
		vec3 R2 = cross( vN, vSigmaX );
		float fDet = dot( vSigmaX, R1 ) * faceDirection;
		vec3 vGrad = sign( fDet ) * ( dHdxy.x * R1 + dHdxy.y * R2 );
		return normalize( abs( fDet ) * surf_norm - vGrad );
	}
#endif`,Xm=`#if NUM_CLIPPING_PLANES > 0
	vec4 plane;
	#ifdef ALPHA_TO_COVERAGE
		float distanceToPlane, distanceGradient;
		float clipOpacity = 1.0;
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
			distanceGradient = fwidth( distanceToPlane ) / 2.0;
			clipOpacity *= smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			if ( clipOpacity == 0.0 ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			float unionClipOpacity = 1.0;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
				distanceGradient = fwidth( distanceToPlane ) / 2.0;
				unionClipOpacity *= 1.0 - smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			}
			#pragma unroll_loop_end
			clipOpacity *= 1.0 - unionClipOpacity;
		#endif
		diffuseColor.a *= clipOpacity;
		if ( diffuseColor.a == 0.0 ) discard;
	#else
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			if ( dot( vClipPosition, plane.xyz ) > plane.w ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			bool clipped = true;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				clipped = ( dot( vClipPosition, plane.xyz ) > plane.w ) && clipped;
			}
			#pragma unroll_loop_end
			if ( clipped ) discard;
		#endif
	#endif
#endif`,qm=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
	uniform vec4 clippingPlanes[ NUM_CLIPPING_PLANES ];
#endif`,Ym=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
#endif`,Zm=`#if NUM_CLIPPING_PLANES > 0
	vClipPosition = - mvPosition.xyz;
#endif`,Km=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	diffuseColor *= vColor;
#endif`,jm=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	varying vec4 vColor;
#endif`,Jm=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	varying vec4 vColor;
#endif`,$m=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	vColor = vec4( 1.0 );
#endif
#ifdef USE_COLOR_ALPHA
	vColor *= color;
#elif defined( USE_COLOR )
	vColor.rgb *= color;
#endif
#ifdef USE_INSTANCING_COLOR
	vColor.rgb *= instanceColor.rgb;
#endif
#ifdef USE_BATCHING_COLOR
	vColor *= getBatchingColor( getIndirectIndex( gl_DrawID ) );
#endif`,Qm=`#define PI 3.141592653589793
#define PI2 6.283185307179586
#define PI_HALF 1.5707963267948966
#define RECIPROCAL_PI 0.3183098861837907
#define RECIPROCAL_PI2 0.15915494309189535
#define EPSILON 1e-6
#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
#define whiteComplement( a ) ( 1.0 - saturate( a ) )
float pow2( const in float x ) { return x*x; }
vec3 pow2( const in vec3 x ) { return x*x; }
float pow3( const in float x ) { return x*x*x; }
float pow4( const in float x ) { float x2 = x*x; return x2*x2; }
float max3( const in vec3 v ) { return max( max( v.x, v.y ), v.z ); }
float average( const in vec3 v ) { return dot( v, vec3( 0.3333333 ) ); }
highp float rand( const in vec2 uv ) {
	const highp float a = 12.9898, b = 78.233, c = 43758.5453;
	highp float dt = dot( uv.xy, vec2( a,b ) ), sn = mod( dt, PI );
	return fract( sin( sn ) * c );
}
#ifdef HIGH_PRECISION
	float precisionSafeLength( vec3 v ) { return length( v ); }
#else
	float precisionSafeLength( vec3 v ) {
		float maxComponent = max3( abs( v ) );
		return length( v / maxComponent ) * maxComponent;
	}
#endif
struct IncidentLight {
	vec3 color;
	vec3 direction;
	bool visible;
};
struct ReflectedLight {
	vec3 directDiffuse;
	vec3 directSpecular;
	vec3 indirectDiffuse;
	vec3 indirectSpecular;
};
#ifdef USE_ALPHAHASH
	varying vec3 vPosition;
#endif
vec3 transformDirection( in vec3 dir, in mat4 matrix ) {
	return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );
}
#define inverseTransformDirection transformDirectionByInverseViewMatrix
vec3 transformNormalByInverseViewMatrix( in vec3 normal, in mat4 viewMatrix ) {
	return normalize( ( vec4( normal, 0.0 ) * viewMatrix ).xyz );
}
vec3 transformDirectionByInverseViewMatrix( in vec3 dir, in mat4 viewMatrix ) {
	return normalize( ( vec4( dir, 0.0 ) * viewMatrix ).xyz );
}
bool isPerspectiveMatrix( mat4 m ) {
	return m[ 2 ][ 3 ] == - 1.0;
}
vec2 equirectUv( in vec3 dir ) {
	float u = atan( dir.z, dir.x ) * RECIPROCAL_PI2 + 0.5;
	float v = asin( clamp( dir.y, - 1.0, 1.0 ) ) * RECIPROCAL_PI + 0.5;
	return vec2( u, v );
}
vec3 BRDF_Lambert( const in vec3 diffuseColor ) {
	return RECIPROCAL_PI * diffuseColor;
}
vec3 F_Schlick( const in vec3 f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
}
float F_Schlick( const in float f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
} // validated`,eg=`#ifdef ENVMAP_TYPE_CUBE_UV
	#define cubeUV_minMipLevel 4.0
	#define cubeUV_minTileSize 16.0
	float getFace( vec3 direction ) {
		vec3 absDirection = abs( direction );
		float face = - 1.0;
		if ( absDirection.x > absDirection.z ) {
			if ( absDirection.x > absDirection.y )
				face = direction.x > 0.0 ? 0.0 : 3.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		} else {
			if ( absDirection.z > absDirection.y )
				face = direction.z > 0.0 ? 2.0 : 5.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		}
		return face;
	}
	vec2 getUV( vec3 direction, float face ) {
		vec2 uv;
		if ( face == 0.0 ) {
			uv = vec2( direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 1.0 ) {
			uv = vec2( - direction.x, - direction.z ) / abs( direction.y );
		} else if ( face == 2.0 ) {
			uv = vec2( - direction.x, direction.y ) / abs( direction.z );
		} else if ( face == 3.0 ) {
			uv = vec2( - direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 4.0 ) {
			uv = vec2( - direction.x, direction.z ) / abs( direction.y );
		} else {
			uv = vec2( direction.x, direction.y ) / abs( direction.z );
		}
		return 0.5 * ( uv + 1.0 );
	}
	vec3 bilinearCubeUV( sampler2D envMap, vec3 direction, float mipInt ) {
		float face = getFace( direction );
		float filterInt = max( cubeUV_minMipLevel - mipInt, 0.0 );
		mipInt = max( mipInt, cubeUV_minMipLevel );
		float faceSize = exp2( mipInt );
		highp vec2 uv = getUV( direction, face ) * ( faceSize - 2.0 ) + 1.0;
		if ( face > 2.0 ) {
			uv.y += faceSize;
			face -= 3.0;
		}
		uv.x += face * faceSize;
		uv.x += filterInt * 3.0 * cubeUV_minTileSize;
		uv.y += 4.0 * ( exp2( CUBEUV_MAX_MIP ) - faceSize );
		uv.x *= CUBEUV_TEXEL_WIDTH;
		uv.y *= CUBEUV_TEXEL_HEIGHT;
		#ifdef texture2DGradEXT
			return texture2DGradEXT( envMap, uv, vec2( 0.0 ), vec2( 0.0 ) ).rgb;
		#else
			return texture2D( envMap, uv ).rgb;
		#endif
	}
	#define cubeUV_r0 1.0
	#define cubeUV_m0 - 2.0
	#define cubeUV_r1 0.8
	#define cubeUV_m1 - 1.0
	#define cubeUV_r4 0.4
	#define cubeUV_m4 2.0
	#define cubeUV_r5 0.305
	#define cubeUV_m5 3.0
	#define cubeUV_r6 0.21
	#define cubeUV_m6 4.0
	float roughnessToMip( float roughness ) {
		float mip = 0.0;
		if ( roughness >= cubeUV_r1 ) {
			mip = ( cubeUV_r0 - roughness ) * ( cubeUV_m1 - cubeUV_m0 ) / ( cubeUV_r0 - cubeUV_r1 ) + cubeUV_m0;
		} else if ( roughness >= cubeUV_r4 ) {
			mip = ( cubeUV_r1 - roughness ) * ( cubeUV_m4 - cubeUV_m1 ) / ( cubeUV_r1 - cubeUV_r4 ) + cubeUV_m1;
		} else if ( roughness >= cubeUV_r5 ) {
			mip = ( cubeUV_r4 - roughness ) * ( cubeUV_m5 - cubeUV_m4 ) / ( cubeUV_r4 - cubeUV_r5 ) + cubeUV_m4;
		} else if ( roughness >= cubeUV_r6 ) {
			mip = ( cubeUV_r5 - roughness ) * ( cubeUV_m6 - cubeUV_m5 ) / ( cubeUV_r5 - cubeUV_r6 ) + cubeUV_m5;
		} else {
			mip = - 2.0 * log2( 1.16 * roughness );		}
		return mip;
	}
	vec4 textureCubeUV( sampler2D envMap, vec3 sampleDir, float roughness ) {
		float mip = clamp( roughnessToMip( roughness ), cubeUV_m0, CUBEUV_MAX_MIP );
		float mipF = fract( mip );
		float mipInt = floor( mip );
		vec3 color0 = bilinearCubeUV( envMap, sampleDir, mipInt );
		if ( mipF == 0.0 ) {
			return vec4( color0, 1.0 );
		} else {
			vec3 color1 = bilinearCubeUV( envMap, sampleDir, mipInt + 1.0 );
			return vec4( mix( color0, color1, mipF ), 1.0 );
		}
	}
#endif`,tg=`vec3 transformedNormal = objectNormal;
#ifdef USE_TANGENT
	vec3 transformedTangent = objectTangent;
#endif
#ifdef USE_BATCHING
	mat3 bm = mat3( batchingMatrix );
	transformedNormal /= vec3( dot( bm[ 0 ], bm[ 0 ] ), dot( bm[ 1 ], bm[ 1 ] ), dot( bm[ 2 ], bm[ 2 ] ) );
	transformedNormal = bm * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = bm * transformedTangent;
	#endif
#endif
#ifdef USE_INSTANCING
	mat3 im = mat3( instanceMatrix );
	transformedNormal /= vec3( dot( im[ 0 ], im[ 0 ] ), dot( im[ 1 ], im[ 1 ] ), dot( im[ 2 ], im[ 2 ] ) );
	transformedNormal = im * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = im * transformedTangent;
	#endif
#endif
transformedNormal = normalMatrix * transformedNormal;
#ifdef FLIP_SIDED
	transformedNormal = - transformedNormal;
#endif
#ifdef USE_TANGENT
	transformedTangent = ( modelViewMatrix * vec4( transformedTangent, 0.0 ) ).xyz;
#endif`,ng=`#ifdef USE_DISPLACEMENTMAP
	uniform sampler2D displacementMap;
	uniform float displacementScale;
	uniform float displacementBias;
#endif`,ig=`#ifdef USE_DISPLACEMENTMAP
	transformed += normalize( objectNormal ) * ( texture2D( displacementMap, vDisplacementMapUv ).x * displacementScale + displacementBias );
#endif`,sg=`#ifdef USE_EMISSIVEMAP
	vec4 emissiveColor = texture2D( emissiveMap, vEmissiveMapUv );
	#ifdef DECODE_VIDEO_TEXTURE_EMISSIVE
		emissiveColor = sRGBTransferEOTF( emissiveColor );
	#endif
	totalEmissiveRadiance *= emissiveColor.rgb;
#endif`,rg=`#ifdef USE_EMISSIVEMAP
	uniform sampler2D emissiveMap;
#endif`,og="gl_FragColor = linearToOutputTexel( gl_FragColor );",ag=`vec4 LinearTransferOETF( in vec4 value ) {
	return value;
}
vec4 sRGBTransferEOTF( in vec4 value ) {
	return vec4( mix( pow( value.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), value.rgb * 0.0773993808, vec3( lessThanEqual( value.rgb, vec3( 0.04045 ) ) ) ), value.a );
}
vec4 sRGBTransferOETF( in vec4 value ) {
	return vec4( mix( pow( value.rgb, vec3( 0.41666 ) ) * 1.055 - vec3( 0.055 ), value.rgb * 12.92, vec3( lessThanEqual( value.rgb, vec3( 0.0031308 ) ) ) ), value.a );
}`,cg=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vec3 cameraToFrag;
		if ( isOrthographic ) {
			cameraToFrag = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToFrag = normalize( vWorldPosition - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vec3 reflectVec = reflect( cameraToFrag, worldNormal );
		#else
			vec3 reflectVec = refract( cameraToFrag, worldNormal, refractionRatio );
		#endif
	#else
		vec3 reflectVec = vReflect;
	#endif
	#ifdef ENVMAP_TYPE_CUBE
		vec4 envColor = textureCube( envMap, envMapRotation * reflectVec );
		#ifdef ENVMAP_BLENDING_MULTIPLY
			outgoingLight = mix( outgoingLight, outgoingLight * envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_MIX )
			outgoingLight = mix( outgoingLight, envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_ADD )
			outgoingLight += envColor.xyz * specularStrength * reflectivity;
		#endif
	#endif
#endif`,lg=`#ifdef USE_ENVMAP
	uniform float envMapIntensity;
	uniform mat3 envMapRotation;
	#ifdef ENVMAP_TYPE_CUBE
		uniform samplerCube envMap;
	#else
		uniform sampler2D envMap;
	#endif
#endif`,hg=`#ifdef USE_ENVMAP
	uniform float reflectivity;
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		varying vec3 vWorldPosition;
		uniform float refractionRatio;
	#else
		varying vec3 vReflect;
	#endif
#endif`,ug=`#ifdef USE_ENVMAP
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		
		varying vec3 vWorldPosition;
	#else
		varying vec3 vReflect;
		uniform float refractionRatio;
	#endif
#endif`,fg=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vWorldPosition = worldPosition.xyz;
	#else
		vec3 cameraToVertex;
		if ( isOrthographic ) {
			cameraToVertex = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToVertex = normalize( worldPosition.xyz - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vReflect = reflect( cameraToVertex, worldNormal );
		#else
			vReflect = refract( cameraToVertex, worldNormal, refractionRatio );
		#endif
	#endif
#endif`,dg=`#ifdef USE_FOG
	vFogDepth = - mvPosition.z;
#endif`,pg=`#ifdef USE_FOG
	varying float vFogDepth;
#endif`,mg=`#ifdef USE_FOG
	#ifdef FOG_EXP2
		float fogFactor = 1.0 - exp( - fogDensity * fogDensity * vFogDepth * vFogDepth );
	#else
		float fogFactor = smoothstep( fogNear, fogFar, vFogDepth );
	#endif
	gl_FragColor.rgb = mix( gl_FragColor.rgb, fogColor, fogFactor );
#endif`,gg=`#ifdef USE_FOG
	uniform vec3 fogColor;
	varying float vFogDepth;
	#ifdef FOG_EXP2
		uniform float fogDensity;
	#else
		uniform float fogNear;
		uniform float fogFar;
	#endif
#endif`,_g=`#ifdef USE_GRADIENTMAP
	uniform sampler2D gradientMap;
#endif
vec3 getGradientIrradiance( vec3 normal, vec3 lightDirection ) {
	float dotNL = dot( normal, lightDirection );
	vec2 coord = vec2( dotNL * 0.5 + 0.5, 0.0 );
	#ifdef USE_GRADIENTMAP
		return vec3( texture2D( gradientMap, coord ).r );
	#else
		vec2 fw = fwidth( coord ) * 0.5;
		return mix( vec3( 0.7 ), vec3( 1.0 ), smoothstep( 0.7 - fw.x, 0.7 + fw.x, coord.x ) );
	#endif
}`,xg=`#ifdef USE_LIGHTMAP
	uniform sampler2D lightMap;
	uniform float lightMapIntensity;
#endif`,yg=`LambertMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularStrength = specularStrength;`,vg=`varying vec3 vViewPosition;
struct LambertMaterial {
	vec3 diffuseColor;
	float specularStrength;
};
void RE_Direct_Lambert( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Lambert( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Lambert
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Lambert`,bg=`uniform bool receiveShadow;
uniform vec3 ambientLightColor;
#if defined( USE_LIGHT_PROBES )
	uniform vec3 lightProbe[ 9 ];
#endif
vec3 shGetIrradianceAt( in vec3 normal, in vec3 shCoefficients[ 9 ] ) {
	float x = normal.x, y = normal.y, z = normal.z;
	vec3 result = shCoefficients[ 0 ] * 0.886227;
	result += shCoefficients[ 1 ] * 2.0 * 0.511664 * y;
	result += shCoefficients[ 2 ] * 2.0 * 0.511664 * z;
	result += shCoefficients[ 3 ] * 2.0 * 0.511664 * x;
	result += shCoefficients[ 4 ] * 2.0 * 0.429043 * x * y;
	result += shCoefficients[ 5 ] * 2.0 * 0.429043 * y * z;
	result += shCoefficients[ 6 ] * ( 0.743125 * z * z - 0.247708 );
	result += shCoefficients[ 7 ] * 2.0 * 0.429043 * x * z;
	result += shCoefficients[ 8 ] * 0.429043 * ( x * x - y * y );
	return result;
}
vec3 getLightProbeIrradiance( const in vec3 lightProbe[ 9 ], const in vec3 normal ) {
	vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec3 irradiance = shGetIrradianceAt( worldNormal, lightProbe );
	return irradiance;
}
vec3 getAmbientLightIrradiance( const in vec3 ambientLightColor ) {
	vec3 irradiance = ambientLightColor;
	return irradiance;
}
float getDistanceAttenuation( const in float lightDistance, const in float cutoffDistance, const in float decayExponent ) {
	float distanceFalloff = 1.0 / max( pow( lightDistance, decayExponent ), 0.01 );
	if ( cutoffDistance > 0.0 ) {
		distanceFalloff *= pow2( saturate( 1.0 - pow4( lightDistance / cutoffDistance ) ) );
	}
	return distanceFalloff;
}
float getSpotAttenuation( const in float coneCosine, const in float penumbraCosine, const in float angleCosine ) {
	return smoothstep( coneCosine, penumbraCosine, angleCosine );
}
#if NUM_DIR_LIGHTS > 0
	struct DirectionalLight {
		vec3 direction;
		vec3 color;
	};
	uniform DirectionalLight directionalLights[ NUM_DIR_LIGHTS ];
	void getDirectionalLightInfo( const in DirectionalLight directionalLight, out IncidentLight light ) {
		light.color = directionalLight.color;
		light.direction = directionalLight.direction;
		light.visible = true;
	}
#endif
#if NUM_POINT_LIGHTS > 0
	struct PointLight {
		vec3 position;
		vec3 color;
		float distance;
		float decay;
	};
	uniform PointLight pointLights[ NUM_POINT_LIGHTS ];
	void getPointLightInfo( const in PointLight pointLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = pointLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float lightDistance = length( lVector );
		light.color = pointLight.color;
		light.color *= getDistanceAttenuation( lightDistance, pointLight.distance, pointLight.decay );
		light.visible = ( light.color != vec3( 0.0 ) );
	}
#endif
#if NUM_SPOT_LIGHTS > 0
	struct SpotLight {
		vec3 position;
		vec3 direction;
		vec3 color;
		float distance;
		float decay;
		float coneCos;
		float penumbraCos;
	};
	uniform SpotLight spotLights[ NUM_SPOT_LIGHTS ];
	void getSpotLightInfo( const in SpotLight spotLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = spotLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float angleCos = dot( light.direction, spotLight.direction );
		float spotAttenuation = getSpotAttenuation( spotLight.coneCos, spotLight.penumbraCos, angleCos );
		if ( spotAttenuation > 0.0 ) {
			float lightDistance = length( lVector );
			light.color = spotLight.color * spotAttenuation;
			light.color *= getDistanceAttenuation( lightDistance, spotLight.distance, spotLight.decay );
			light.visible = ( light.color != vec3( 0.0 ) );
		} else {
			light.color = vec3( 0.0 );
			light.visible = false;
		}
	}
#endif
#if NUM_RECT_AREA_LIGHTS > 0
	struct RectAreaLight {
		vec3 color;
		vec3 position;
		vec3 halfWidth;
		vec3 halfHeight;
	};
	uniform sampler2D ltc_1;	uniform sampler2D ltc_2;
	uniform RectAreaLight rectAreaLights[ NUM_RECT_AREA_LIGHTS ];
#endif
#if NUM_HEMI_LIGHTS > 0
	struct HemisphereLight {
		vec3 direction;
		vec3 skyColor;
		vec3 groundColor;
	};
	uniform HemisphereLight hemisphereLights[ NUM_HEMI_LIGHTS ];
	vec3 getHemisphereLightIrradiance( const in HemisphereLight hemiLight, const in vec3 normal ) {
		float dotNL = dot( normal, hemiLight.direction );
		float hemiDiffuseWeight = 0.5 * dotNL + 0.5;
		vec3 irradiance = mix( hemiLight.groundColor, hemiLight.skyColor, hemiDiffuseWeight );
		return irradiance;
	}
#endif
#include <lightprobes_pars_fragment>`,Mg=`#ifdef USE_ENVMAP
	vec3 getIBLIrradiance( const in vec3 normal ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * worldNormal, 1.0 );
			return PI * envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	vec3 getIBLRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 reflectVec = reflect( - viewDir, normal );
			reflectVec = normalize( mix( reflectVec, normal, pow4( roughness ) ) );
			reflectVec = transformDirectionByInverseViewMatrix( reflectVec, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * reflectVec, roughness );
			return envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	#ifdef USE_ANISOTROPY
		vec3 getIBLAnisotropyRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness, const in vec3 bitangent, const in float anisotropy ) {
			#ifdef ENVMAP_TYPE_CUBE_UV
				vec3 bentNormal = cross( bitangent, viewDir );
				bentNormal = normalize( cross( bentNormal, bitangent ) );
				bentNormal = normalize( mix( bentNormal, normal, pow2( pow2( 1.0 - anisotropy * ( 1.0 - roughness ) ) ) ) );
				return getIBLRadiance( viewDir, bentNormal, roughness );
			#else
				return vec3( 0.0 );
			#endif
		}
	#endif
#endif`,Sg=`ToonMaterial material;
material.diffuseColor = diffuseColor.rgb;`,Tg=`varying vec3 vViewPosition;
struct ToonMaterial {
	vec3 diffuseColor;
};
void RE_Direct_Toon( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 irradiance = getGradientIrradiance( geometryNormal, directLight.direction ) * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Toon( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Toon
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Toon`,Eg=`BlinnPhongMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularColor = specular;
material.specularShininess = shininess;
material.specularStrength = specularStrength;`,Ag=`varying vec3 vViewPosition;
struct BlinnPhongMaterial {
	vec3 diffuseColor;
	vec3 specularColor;
	float specularShininess;
	float specularStrength;
};
void RE_Direct_BlinnPhong( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
	reflectedLight.directSpecular += irradiance * BRDF_BlinnPhong( directLight.direction, geometryViewDir, geometryNormal, material.specularColor, material.specularShininess ) * material.specularStrength;
}
void RE_IndirectDiffuse_BlinnPhong( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_BlinnPhong
#define RE_IndirectDiffuse		RE_IndirectDiffuse_BlinnPhong`,wg=`PhysicalMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.diffuseContribution = diffuseColor.rgb * ( 1.0 - metalnessFactor );
material.metalness = metalnessFactor;
vec3 dxy = max( abs( dFdx( nonPerturbedNormal ) ), abs( dFdy( nonPerturbedNormal ) ) );
float geometryRoughness = max( max( dxy.x, dxy.y ), dxy.z );
material.roughness = max( roughnessFactor, 0.0525 );material.roughness += geometryRoughness;
material.roughness = min( material.roughness, 1.0 );
#ifdef IOR
	material.ior = ior;
	#ifdef USE_SPECULAR
		float specularIntensityFactor = specularIntensity;
		vec3 specularColorFactor = specularColor;
		#ifdef USE_SPECULAR_COLORMAP
			specularColorFactor *= texture2D( specularColorMap, vSpecularColorMapUv ).rgb;
		#endif
		#ifdef USE_SPECULAR_INTENSITYMAP
			specularIntensityFactor *= texture2D( specularIntensityMap, vSpecularIntensityMapUv ).a;
		#endif
		material.specularF90 = mix( specularIntensityFactor, 1.0, metalnessFactor );
	#else
		float specularIntensityFactor = 1.0;
		vec3 specularColorFactor = vec3( 1.0 );
		material.specularF90 = 1.0;
	#endif
	material.specularColor = min( pow2( ( material.ior - 1.0 ) / ( material.ior + 1.0 ) ) * specularColorFactor, vec3( 1.0 ) ) * specularIntensityFactor;
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
#else
	material.specularColor = vec3( 0.04 );
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
	material.specularF90 = 1.0;
#endif
#ifdef USE_CLEARCOAT
	material.clearcoat = clearcoat;
	material.clearcoatRoughness = clearcoatRoughness;
	material.clearcoatF0 = vec3( 0.04 );
	material.clearcoatF90 = 1.0;
	#ifdef USE_CLEARCOATMAP
		material.clearcoat *= texture2D( clearcoatMap, vClearcoatMapUv ).x;
	#endif
	#ifdef USE_CLEARCOAT_ROUGHNESSMAP
		material.clearcoatRoughness *= texture2D( clearcoatRoughnessMap, vClearcoatRoughnessMapUv ).y;
	#endif
	material.clearcoat = saturate( material.clearcoat );	material.clearcoatRoughness = max( material.clearcoatRoughness, 0.0525 );
	material.clearcoatRoughness += geometryRoughness;
	material.clearcoatRoughness = min( material.clearcoatRoughness, 1.0 );
#endif
#ifdef USE_DISPERSION
	material.dispersion = dispersion;
#endif
#ifdef USE_IRIDESCENCE
	material.iridescence = iridescence;
	material.iridescenceIOR = iridescenceIOR;
	#ifdef USE_IRIDESCENCEMAP
		material.iridescence *= texture2D( iridescenceMap, vIridescenceMapUv ).r;
	#endif
	#ifdef USE_IRIDESCENCE_THICKNESSMAP
		material.iridescenceThickness = (iridescenceThicknessMaximum - iridescenceThicknessMinimum) * texture2D( iridescenceThicknessMap, vIridescenceThicknessMapUv ).g + iridescenceThicknessMinimum;
	#else
		material.iridescenceThickness = iridescenceThicknessMaximum;
	#endif
#endif
#ifdef USE_SHEEN
	material.sheenColor = sheenColor;
	#ifdef USE_SHEEN_COLORMAP
		material.sheenColor *= texture2D( sheenColorMap, vSheenColorMapUv ).rgb;
	#endif
	material.sheenRoughness = clamp( sheenRoughness, 0.0001, 1.0 );
	#ifdef USE_SHEEN_ROUGHNESSMAP
		material.sheenRoughness *= texture2D( sheenRoughnessMap, vSheenRoughnessMapUv ).a;
	#endif
#endif
#ifdef USE_ANISOTROPY
	#ifdef USE_ANISOTROPYMAP
		mat2 anisotropyMat = mat2( anisotropyVector.x, anisotropyVector.y, - anisotropyVector.y, anisotropyVector.x );
		vec3 anisotropyPolar = texture2D( anisotropyMap, vAnisotropyMapUv ).rgb;
		vec2 anisotropyV = anisotropyMat * normalize( 2.0 * anisotropyPolar.rg - vec2( 1.0 ) ) * anisotropyPolar.b;
	#else
		vec2 anisotropyV = anisotropyVector;
	#endif
	material.anisotropy = length( anisotropyV );
	if( material.anisotropy == 0.0 ) {
		anisotropyV = vec2( 1.0, 0.0 );
	} else {
		anisotropyV /= material.anisotropy;
		material.anisotropy = saturate( material.anisotropy );
	}
	material.alphaT = mix( pow2( material.roughness ), 1.0, pow2( material.anisotropy ) );
	material.anisotropyT = tbn[ 0 ] * anisotropyV.x + tbn[ 1 ] * anisotropyV.y;
	material.anisotropyB = tbn[ 1 ] * anisotropyV.x - tbn[ 0 ] * anisotropyV.y;
#endif`,Rg=`uniform sampler2D dfgLUT;
struct PhysicalMaterial {
	vec3 diffuseColor;
	vec3 diffuseContribution;
	vec3 specularColor;
	vec3 specularColorBlended;
	float roughness;
	float metalness;
	float specularF90;
	float dispersion;
	#ifdef USE_CLEARCOAT
		float clearcoat;
		float clearcoatRoughness;
		vec3 clearcoatF0;
		float clearcoatF90;
	#endif
	#ifdef USE_IRIDESCENCE
		float iridescence;
		float iridescenceIOR;
		float iridescenceThickness;
		vec3 iridescenceFresnel;
		vec3 iridescenceF0;
		vec3 iridescenceFresnelDielectric;
		vec3 iridescenceFresnelMetallic;
	#endif
	#ifdef USE_SHEEN
		vec3 sheenColor;
		float sheenRoughness;
	#endif
	#ifdef IOR
		float ior;
	#endif
	#ifdef USE_TRANSMISSION
		float transmission;
		float transmissionAlpha;
		float thickness;
		float attenuationDistance;
		vec3 attenuationColor;
	#endif
	#ifdef USE_ANISOTROPY
		float anisotropy;
		float alphaT;
		vec3 anisotropyT;
		vec3 anisotropyB;
	#endif
};
vec3 clearcoatSpecularDirect = vec3( 0.0 );
vec3 clearcoatSpecularIndirect = vec3( 0.0 );
vec3 sheenSpecularDirect = vec3( 0.0 );
vec3 sheenSpecularIndirect = vec3(0.0 );
vec3 Schlick_to_F0( const in vec3 f, const in float f90, const in float dotVH ) {
    float x = clamp( 1.0 - dotVH, 0.0, 1.0 );
    float x2 = x * x;
    float x5 = clamp( x * x2 * x2, 0.0, 0.9999 );
    return ( f - vec3( f90 ) * x5 ) / ( 1.0 - x5 );
}
float V_GGX_SmithCorrelated( const in float alpha, const in float dotNL, const in float dotNV ) {
	float a2 = pow2( alpha );
	float gv = dotNL * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNV ) );
	float gl = dotNV * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNL ) );
	return 0.5 / max( gv + gl, EPSILON );
}
float D_GGX( const in float alpha, const in float dotNH ) {
	float a2 = pow2( alpha );
	float denom = pow2( dotNH ) * ( a2 - 1.0 ) + 1.0;
	return RECIPROCAL_PI * a2 / pow2( denom );
}
#ifdef USE_ANISOTROPY
	float V_GGX_SmithCorrelated_Anisotropic( const in float alphaT, const in float alphaB, const in float dotTV, const in float dotBV, const in float dotTL, const in float dotBL, const in float dotNV, const in float dotNL ) {
		float gv = dotNL * length( vec3( alphaT * dotTV, alphaB * dotBV, dotNV ) );
		float gl = dotNV * length( vec3( alphaT * dotTL, alphaB * dotBL, dotNL ) );
		return 0.5 / max( gv + gl, EPSILON );
	}
	float D_GGX_Anisotropic( const in float alphaT, const in float alphaB, const in float dotNH, const in float dotTH, const in float dotBH ) {
		float a2 = alphaT * alphaB;
		highp vec3 v = vec3( alphaB * dotTH, alphaT * dotBH, a2 * dotNH );
		highp float v2 = dot( v, v );
		float w2 = a2 / v2;
		return RECIPROCAL_PI * a2 * pow2 ( w2 );
	}
#endif
#ifdef USE_CLEARCOAT
	vec3 BRDF_GGX_Clearcoat( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material) {
		vec3 f0 = material.clearcoatF0;
		float f90 = material.clearcoatF90;
		float roughness = material.clearcoatRoughness;
		float alpha = pow2( roughness );
		vec3 halfDir = normalize( lightDir + viewDir );
		float dotNL = saturate( dot( normal, lightDir ) );
		float dotNV = saturate( dot( normal, viewDir ) );
		float dotNH = saturate( dot( normal, halfDir ) );
		float dotVH = saturate( dot( viewDir, halfDir ) );
		vec3 F = F_Schlick( f0, f90, dotVH );
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
		return F * ( V * D );
	}
#endif
vec3 BRDF_GGX( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material ) {
	vec3 f0 = material.specularColorBlended;
	float f90 = material.specularF90;
	float roughness = material.roughness;
	float alpha = pow2( roughness );
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( f0, f90, dotVH );
	#ifdef USE_IRIDESCENCE
		F = mix( F, material.iridescenceFresnel, material.iridescence );
	#endif
	#ifdef USE_ANISOTROPY
		float dotTL = dot( material.anisotropyT, lightDir );
		float dotTV = dot( material.anisotropyT, viewDir );
		float dotTH = dot( material.anisotropyT, halfDir );
		float dotBL = dot( material.anisotropyB, lightDir );
		float dotBV = dot( material.anisotropyB, viewDir );
		float dotBH = dot( material.anisotropyB, halfDir );
		float V = V_GGX_SmithCorrelated_Anisotropic( material.alphaT, alpha, dotTV, dotBV, dotTL, dotBL, dotNV, dotNL );
		float D = D_GGX_Anisotropic( material.alphaT, alpha, dotNH, dotTH, dotBH );
	#else
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
	#endif
	return F * ( V * D );
}
vec2 LTC_Uv( const in vec3 N, const in vec3 V, const in float roughness ) {
	const float LUT_SIZE = 64.0;
	const float LUT_SCALE = ( LUT_SIZE - 1.0 ) / LUT_SIZE;
	const float LUT_BIAS = 0.5 / LUT_SIZE;
	float dotNV = saturate( dot( N, V ) );
	vec2 uv = vec2( roughness, sqrt( 1.0 - dotNV ) );
	uv = uv * LUT_SCALE + LUT_BIAS;
	return uv;
}
float LTC_ClippedSphereFormFactor( const in vec3 f ) {
	float l = length( f );
	return max( ( l * l + f.z ) / ( l + 1.0 ), 0.0 );
}
vec3 LTC_EdgeVectorFormFactor( const in vec3 v1, const in vec3 v2 ) {
	float x = dot( v1, v2 );
	float y = abs( x );
	float a = 0.8543985 + ( 0.4965155 + 0.0145206 * y ) * y;
	float b = 3.4175940 + ( 4.1616724 + y ) * y;
	float v = a / b;
	float theta_sintheta = ( x > 0.0 ) ? v : 0.5 * inversesqrt( max( 1.0 - x * x, 1e-7 ) ) - v;
	return cross( v1, v2 ) * theta_sintheta;
}
vec3 LTC_Evaluate( const in vec3 N, const in vec3 V, const in vec3 P, const in mat3 mInv, const in vec3 rectCoords[ 4 ] ) {
	vec3 v1 = rectCoords[ 1 ] - rectCoords[ 0 ];
	vec3 v2 = rectCoords[ 3 ] - rectCoords[ 0 ];
	vec3 lightNormal = cross( v1, v2 );
	if( dot( lightNormal, P - rectCoords[ 0 ] ) < 0.0 ) return vec3( 0.0 );
	vec3 T1, T2;
	T1 = normalize( V - N * dot( V, N ) );
	T2 = - cross( N, T1 );
	mat3 mat = mInv * transpose( mat3( T1, T2, N ) );
	vec3 coords[ 4 ];
	coords[ 0 ] = mat * ( rectCoords[ 0 ] - P );
	coords[ 1 ] = mat * ( rectCoords[ 1 ] - P );
	coords[ 2 ] = mat * ( rectCoords[ 2 ] - P );
	coords[ 3 ] = mat * ( rectCoords[ 3 ] - P );
	coords[ 0 ] = normalize( coords[ 0 ] );
	coords[ 1 ] = normalize( coords[ 1 ] );
	coords[ 2 ] = normalize( coords[ 2 ] );
	coords[ 3 ] = normalize( coords[ 3 ] );
	vec3 vectorFormFactor = vec3( 0.0 );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 0 ], coords[ 1 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 1 ], coords[ 2 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 2 ], coords[ 3 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 3 ], coords[ 0 ] );
	float result = LTC_ClippedSphereFormFactor( vectorFormFactor );
	return vec3( result );
}
#if defined( USE_SHEEN )
float D_Charlie( float roughness, float dotNH ) {
	float alpha = pow2( roughness );
	float invAlpha = 1.0 / alpha;
	float cos2h = dotNH * dotNH;
	float sin2h = max( 1.0 - cos2h, 0.0078125 );
	return ( 2.0 + invAlpha ) * pow( sin2h, invAlpha * 0.5 ) / ( 2.0 * PI );
}
float V_Neubelt( float dotNV, float dotNL ) {
	return saturate( 1.0 / ( 4.0 * ( dotNL + dotNV - dotNL * dotNV ) ) );
}
vec3 BRDF_Sheen( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, vec3 sheenColor, const in float sheenRoughness ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float D = D_Charlie( sheenRoughness, dotNH );
	float V = V_Neubelt( dotNV, dotNL );
	return sheenColor * ( D * V );
}
#endif
float IBLSheenBRDF( const in vec3 normal, const in vec3 viewDir, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	float r2 = roughness * roughness;
	float rInv = 1.0 / ( roughness + 0.1 );
	float a = -1.9362 + 1.0678 * roughness + 0.4573 * r2 - 0.8469 * rInv;
	float b = -0.6014 + 0.5538 * roughness - 0.4670 * r2 - 0.1255 * rInv;
	float DG = exp( a * dotNV + b );
	return saturate( DG );
}
vec3 EnvironmentBRDF( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 fab = texture2D( dfgLUT, vec2( roughness, dotNV ) ).rg;
	return specularColor * fab.x + specularF90 * fab.y;
}
#ifdef USE_IRIDESCENCE
void computeMultiscatteringIridescence( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float iridescence, const in vec3 iridescenceF0, const in float roughness, inout vec3 singleScatter, inout vec3 multiScatter ) {
#else
void computeMultiscattering( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float roughness, inout vec3 singleScatter, inout vec3 multiScatter ) {
#endif
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 fab = texture2D( dfgLUT, vec2( roughness, dotNV ) ).rg;
	#ifdef USE_IRIDESCENCE
		vec3 Fr = mix( specularColor, iridescenceF0, iridescence );
	#else
		vec3 Fr = specularColor;
	#endif
	vec3 FssEss = Fr * fab.x + specularF90 * fab.y;
	float Ess = fab.x + fab.y;
	float Ems = 1.0 - Ess;
	vec3 Favg = Fr + ( 1.0 - Fr ) * 0.047619;	vec3 Fms = FssEss * Favg / ( 1.0 - Ems * Favg );
	singleScatter += FssEss;
	multiScatter += Fms * Ems;
}
vec3 BRDF_GGX_Multiscatter( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material ) {
	vec3 singleScatter = BRDF_GGX( lightDir, viewDir, normal, material );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 dfgV = texture2D( dfgLUT, vec2( material.roughness, dotNV ) ).rg;
	vec2 dfgL = texture2D( dfgLUT, vec2( material.roughness, dotNL ) ).rg;
	vec3 FssEss_V = material.specularColorBlended * dfgV.x + material.specularF90 * dfgV.y;
	vec3 FssEss_L = material.specularColorBlended * dfgL.x + material.specularF90 * dfgL.y;
	float Ess_V = dfgV.x + dfgV.y;
	float Ess_L = dfgL.x + dfgL.y;
	float Ems_V = 1.0 - Ess_V;
	float Ems_L = 1.0 - Ess_L;
	vec3 Favg = material.specularColorBlended + ( 1.0 - material.specularColorBlended ) * 0.047619;
	vec3 Fms = FssEss_V * FssEss_L * Favg / ( 1.0 - Ems_V * Ems_L * Favg + EPSILON );
	float compensationFactor = Ems_V * Ems_L;
	vec3 multiScatter = Fms * compensationFactor;
	return singleScatter + multiScatter;
}
#if NUM_RECT_AREA_LIGHTS > 0
	void RE_Direct_RectArea_Physical( const in RectAreaLight rectAreaLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
		vec3 normal = geometryNormal;
		vec3 viewDir = geometryViewDir;
		vec3 position = geometryPosition;
		vec3 lightPos = rectAreaLight.position;
		vec3 halfWidth = rectAreaLight.halfWidth;
		vec3 halfHeight = rectAreaLight.halfHeight;
		vec3 lightColor = rectAreaLight.color;
		float roughness = material.roughness;
		vec3 rectCoords[ 4 ];
		rectCoords[ 0 ] = lightPos + halfWidth - halfHeight;		rectCoords[ 1 ] = lightPos - halfWidth - halfHeight;
		rectCoords[ 2 ] = lightPos - halfWidth + halfHeight;
		rectCoords[ 3 ] = lightPos + halfWidth + halfHeight;
		vec2 uv = LTC_Uv( normal, viewDir, roughness );
		vec4 t1 = texture2D( ltc_1, uv );
		vec4 t2 = texture2D( ltc_2, uv );
		mat3 mInv = mat3(
			vec3( t1.x, 0, t1.y ),
			vec3(    0, 1,    0 ),
			vec3( t1.z, 0, t1.w )
		);
		vec3 fresnel = ( material.specularColorBlended * t2.x + ( material.specularF90 - material.specularColorBlended ) * t2.y );
		reflectedLight.directSpecular += lightColor * fresnel * LTC_Evaluate( normal, viewDir, position, mInv, rectCoords );
		reflectedLight.directDiffuse += lightColor * material.diffuseContribution * LTC_Evaluate( normal, viewDir, position, mat3( 1.0 ), rectCoords );
		#ifdef USE_CLEARCOAT
			vec3 Ncc = geometryClearcoatNormal;
			vec2 uvClearcoat = LTC_Uv( Ncc, viewDir, material.clearcoatRoughness );
			vec4 t1Clearcoat = texture2D( ltc_1, uvClearcoat );
			vec4 t2Clearcoat = texture2D( ltc_2, uvClearcoat );
			mat3 mInvClearcoat = mat3(
				vec3( t1Clearcoat.x, 0, t1Clearcoat.y ),
				vec3(             0, 1,             0 ),
				vec3( t1Clearcoat.z, 0, t1Clearcoat.w )
			);
			vec3 fresnelClearcoat = material.clearcoatF0 * t2Clearcoat.x + ( material.clearcoatF90 - material.clearcoatF0 ) * t2Clearcoat.y;
			clearcoatSpecularDirect += lightColor * fresnelClearcoat * LTC_Evaluate( Ncc, viewDir, position, mInvClearcoat, rectCoords );
		#endif
	}
#endif
void RE_Direct_Physical( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	#ifdef USE_CLEARCOAT
		float dotNLcc = saturate( dot( geometryClearcoatNormal, directLight.direction ) );
		vec3 ccIrradiance = dotNLcc * directLight.color;
		clearcoatSpecularDirect += ccIrradiance * BRDF_GGX_Clearcoat( directLight.direction, geometryViewDir, geometryClearcoatNormal, material );
	#endif
	#ifdef USE_SHEEN
 
 		sheenSpecularDirect += irradiance * BRDF_Sheen( directLight.direction, geometryViewDir, geometryNormal, material.sheenColor, material.sheenRoughness );
 
 		float sheenAlbedoV = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
 		float sheenAlbedoL = IBLSheenBRDF( geometryNormal, directLight.direction, material.sheenRoughness );
 
 		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * max( sheenAlbedoV, sheenAlbedoL );
 
 		irradiance *= sheenEnergyComp;
 
 	#endif
	reflectedLight.directSpecular += irradiance * BRDF_GGX_Multiscatter( directLight.direction, geometryViewDir, geometryNormal, material );
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseContribution );
}
void RE_IndirectDiffuse_Physical( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 diffuse = irradiance * BRDF_Lambert( material.diffuseContribution );
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		diffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectDiffuse += diffuse;
}
void RE_IndirectSpecular_Physical( const in vec3 radiance, const in vec3 irradiance, const in vec3 clearcoatRadiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight) {
	#ifdef USE_CLEARCOAT
		clearcoatSpecularIndirect += clearcoatRadiance * EnvironmentBRDF( geometryClearcoatNormal, geometryViewDir, material.clearcoatF0, material.clearcoatF90, material.clearcoatRoughness );
	#endif
	#ifdef USE_SHEEN
		sheenSpecularIndirect += irradiance * material.sheenColor * IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness ) * RECIPROCAL_PI;
 	#endif
	vec3 singleScatteringDielectric = vec3( 0.0 );
	vec3 multiScatteringDielectric = vec3( 0.0 );
	vec3 singleScatteringMetallic = vec3( 0.0 );
	vec3 multiScatteringMetallic = vec3( 0.0 );
	#ifdef USE_IRIDESCENCE
		computeMultiscatteringIridescence( geometryNormal, geometryViewDir, material.specularColor, material.specularF90, material.iridescence, material.iridescenceFresnelDielectric, material.roughness, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscatteringIridescence( geometryNormal, geometryViewDir, material.diffuseColor, material.specularF90, material.iridescence, material.iridescenceFresnelMetallic, material.roughness, singleScatteringMetallic, multiScatteringMetallic );
	#else
		computeMultiscattering( geometryNormal, geometryViewDir, material.specularColor, material.specularF90, material.roughness, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscattering( geometryNormal, geometryViewDir, material.diffuseColor, material.specularF90, material.roughness, singleScatteringMetallic, multiScatteringMetallic );
	#endif
	vec3 singleScattering = mix( singleScatteringDielectric, singleScatteringMetallic, material.metalness );
	vec3 multiScattering = mix( multiScatteringDielectric, multiScatteringMetallic, material.metalness );
	vec3 totalScatteringDielectric = singleScatteringDielectric + multiScatteringDielectric;
	vec3 diffuse = material.diffuseContribution * ( 1.0 - totalScatteringDielectric );
	vec3 cosineWeightedIrradiance = irradiance * RECIPROCAL_PI;
	vec3 indirectSpecular = radiance * singleScattering;
	indirectSpecular += multiScattering * cosineWeightedIrradiance;
	vec3 indirectDiffuse = diffuse * cosineWeightedIrradiance;
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		indirectSpecular *= sheenEnergyComp;
		indirectDiffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectSpecular += indirectSpecular;
	reflectedLight.indirectDiffuse += indirectDiffuse;
}
#define RE_Direct				RE_Direct_Physical
#define RE_Direct_RectArea		RE_Direct_RectArea_Physical
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Physical
#define RE_IndirectSpecular		RE_IndirectSpecular_Physical
float computeSpecularOcclusion( const in float dotNV, const in float ambientOcclusion, const in float roughness ) {
	return saturate( pow( dotNV + ambientOcclusion, exp2( - 16.0 * roughness - 1.0 ) ) - 1.0 + ambientOcclusion );
}`,Cg=`
vec3 geometryPosition = - vViewPosition;
vec3 geometryNormal = normal;
vec3 geometryViewDir = ( isOrthographic ) ? vec3( 0, 0, 1 ) : normalize( vViewPosition );
vec3 geometryClearcoatNormal = vec3( 0.0 );
#ifdef USE_CLEARCOAT
	geometryClearcoatNormal = clearcoatNormal;
#endif
#ifdef USE_IRIDESCENCE
	float dotNVi = saturate( dot( normal, geometryViewDir ) );
	if ( material.iridescenceThickness == 0.0 ) {
		material.iridescence = 0.0;
	} else {
		material.iridescence = saturate( material.iridescence );
	}
	if ( material.iridescence > 0.0 ) {
		material.iridescenceFresnelDielectric = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.specularColor );
		material.iridescenceFresnelMetallic = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.diffuseColor );
		material.iridescenceFresnel = mix( material.iridescenceFresnelDielectric, material.iridescenceFresnelMetallic, material.metalness );
		material.iridescenceF0 = Schlick_to_F0( material.iridescenceFresnel, 1.0, dotNVi );
	}
#endif
IncidentLight directLight;
#if ( NUM_POINT_LIGHTS > 0 ) && defined( RE_Direct )
	PointLight pointLight;
	#if defined( USE_SHADOWMAP ) && NUM_POINT_LIGHT_SHADOWS > 0
	PointLightShadow pointLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHTS; i ++ ) {
		pointLight = pointLights[ i ];
		getPointLightInfo( pointLight, geometryPosition, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_POINT_LIGHT_SHADOWS ) && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
		pointLightShadow = pointLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getPointShadow( pointShadowMap[ i ], pointLightShadow.shadowMapSize, pointLightShadow.shadowIntensity, pointLightShadow.shadowBias, pointLightShadow.shadowRadius, vPointShadowCoord[ i ], pointLightShadow.shadowCameraNear, pointLightShadow.shadowCameraFar ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_SPOT_LIGHTS > 0 ) && defined( RE_Direct )
	SpotLight spotLight;
	vec4 spotColor;
	vec3 spotLightCoord;
	bool inSpotLightMap;
	#if defined( USE_SHADOWMAP ) && NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHTS; i ++ ) {
		spotLight = spotLights[ i ];
		getSpotLightInfo( spotLight, geometryPosition, directLight );
		#if ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#define SPOT_LIGHT_MAP_INDEX UNROLLED_LOOP_INDEX
		#elif ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		#define SPOT_LIGHT_MAP_INDEX NUM_SPOT_LIGHT_MAPS
		#else
		#define SPOT_LIGHT_MAP_INDEX ( UNROLLED_LOOP_INDEX - NUM_SPOT_LIGHT_SHADOWS + NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#endif
		#if ( SPOT_LIGHT_MAP_INDEX < NUM_SPOT_LIGHT_MAPS )
			spotLightCoord = vSpotLightCoord[ i ].xyz / vSpotLightCoord[ i ].w;
			inSpotLightMap = all( lessThan( abs( spotLightCoord * 2. - 1. ), vec3( 1.0 ) ) );
			spotColor = texture2D( spotLightMap[ SPOT_LIGHT_MAP_INDEX ], spotLightCoord.xy );
			directLight.color = inSpotLightMap ? directLight.color * spotColor.rgb : directLight.color;
		#endif
		#undef SPOT_LIGHT_MAP_INDEX
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		spotLightShadow = spotLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( spotShadowMap[ i ], spotLightShadow.shadowMapSize, spotLightShadow.shadowIntensity, spotLightShadow.shadowBias, spotLightShadow.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_DIR_LIGHTS > 0 ) && defined( RE_Direct )
	DirectionalLight directionalLight;
	#if defined( USE_SHADOWMAP ) && NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHTS; i ++ ) {
		directionalLight = directionalLights[ i ];
		getDirectionalLightInfo( directionalLight, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_DIR_LIGHT_SHADOWS )
		directionalLightShadow = directionalLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( directionalShadowMap[ i ], directionalLightShadow.shadowMapSize, directionalLightShadow.shadowIntensity, directionalLightShadow.shadowBias, directionalLightShadow.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_RECT_AREA_LIGHTS > 0 ) && defined( RE_Direct_RectArea )
	RectAreaLight rectAreaLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_RECT_AREA_LIGHTS; i ++ ) {
		rectAreaLight = rectAreaLights[ i ];
		RE_Direct_RectArea( rectAreaLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if defined( RE_IndirectDiffuse )
	vec3 iblIrradiance = vec3( 0.0 );
	vec3 irradiance = getAmbientLightIrradiance( ambientLightColor );
	#if defined( USE_LIGHT_PROBES )
		irradiance += getLightProbeIrradiance( lightProbe, geometryNormal );
	#endif
	#if ( NUM_HEMI_LIGHTS > 0 )
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_HEMI_LIGHTS; i ++ ) {
			irradiance += getHemisphereLightIrradiance( hemisphereLights[ i ], geometryNormal );
		}
		#pragma unroll_loop_end
	#endif
	#ifdef USE_LIGHT_PROBES_GRID
		vec3 probeWorldPos = ( ( vec4( geometryPosition, 1.0 ) - viewMatrix[ 3 ] ) * viewMatrix ).xyz;
		vec3 probeWorldNormal = transformNormalByInverseViewMatrix( geometryNormal, viewMatrix );
		irradiance += getLightProbeGridIrradiance( probeWorldPos, probeWorldNormal );
	#endif
#endif
#if defined( RE_IndirectSpecular )
	vec3 radiance = vec3( 0.0 );
	vec3 clearcoatRadiance = vec3( 0.0 );
#endif`,Pg=`#if defined( RE_IndirectDiffuse )
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		vec3 lightMapIrradiance = lightMapTexel.rgb * lightMapIntensity;
		irradiance += lightMapIrradiance;
	#endif
	#if defined( USE_ENVMAP ) && defined( ENVMAP_TYPE_CUBE_UV )
		#if defined( STANDARD ) || defined( LAMBERT ) || defined( PHONG )
			iblIrradiance += getIBLIrradiance( geometryNormal );
		#endif
	#endif
#endif
#if defined( USE_ENVMAP ) && defined( RE_IndirectSpecular )
	#ifdef USE_ANISOTROPY
		radiance += getIBLAnisotropyRadiance( geometryViewDir, geometryNormal, material.roughness, material.anisotropyB, material.anisotropy );
	#else
		radiance += getIBLRadiance( geometryViewDir, geometryNormal, material.roughness );
	#endif
	#ifdef USE_CLEARCOAT
		clearcoatRadiance += getIBLRadiance( geometryViewDir, geometryClearcoatNormal, material.clearcoatRoughness );
	#endif
#endif`,Ig=`#if defined( RE_IndirectDiffuse )
	#if defined( LAMBERT ) || defined( PHONG )
		irradiance += iblIrradiance;
	#endif
	RE_IndirectDiffuse( irradiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif
#if defined( RE_IndirectSpecular )
	RE_IndirectSpecular( radiance, iblIrradiance, clearcoatRadiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif`,Lg=`#ifdef USE_LIGHT_PROBES_GRID
uniform highp sampler3D probesSH;
uniform vec3 probesMin;
uniform vec3 probesMax;
uniform vec3 probesResolution;
vec3 getLightProbeGridIrradiance( vec3 worldPos, vec3 worldNormal ) {
	vec3 res = probesResolution;
	vec3 gridRange = probesMax - probesMin;
	vec3 resMinusOne = res - 1.0;
	vec3 probeSpacing = gridRange / resMinusOne;
	vec3 samplePos = worldPos + worldNormal * probeSpacing * 0.5;
	vec3 uvw = clamp( ( samplePos - probesMin ) / gridRange, 0.0, 1.0 );
	uvw = uvw * resMinusOne / res + 0.5 / res;
	float nz          = res.z;
	float paddedSlices = nz + 2.0;
	float atlasDepth  = 7.0 * paddedSlices;
	float uvZBase     = uvw.z * nz + 1.0;
	vec4 s0 = texture( probesSH, vec3( uvw.xy, ( uvZBase                       ) / atlasDepth ) );
	vec4 s1 = texture( probesSH, vec3( uvw.xy, ( uvZBase +       paddedSlices   ) / atlasDepth ) );
	vec4 s2 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 2.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s3 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 3.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s4 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 4.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s5 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 5.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s6 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 6.0 * paddedSlices   ) / atlasDepth ) );
	vec3 c0 = s0.xyz;
	vec3 c1 = vec3( s0.w, s1.xy );
	vec3 c2 = vec3( s1.zw, s2.x );
	vec3 c3 = s2.yzw;
	vec3 c4 = s3.xyz;
	vec3 c5 = vec3( s3.w, s4.xy );
	vec3 c6 = vec3( s4.zw, s5.x );
	vec3 c7 = s5.yzw;
	vec3 c8 = s6.xyz;
	float x = worldNormal.x, y = worldNormal.y, z = worldNormal.z;
	vec3 result = c0 * 0.886227;
	result += c1 * 2.0 * 0.511664 * y;
	result += c2 * 2.0 * 0.511664 * z;
	result += c3 * 2.0 * 0.511664 * x;
	result += c4 * 2.0 * 0.429043 * x * y;
	result += c5 * 2.0 * 0.429043 * y * z;
	result += c6 * ( 0.743125 * z * z - 0.247708 );
	result += c7 * 2.0 * 0.429043 * x * z;
	result += c8 * 0.429043 * ( x * x - y * y );
	return max( result, vec3( 0.0 ) );
}
#endif`,Dg=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	gl_FragDepth = vIsPerspective == 0.0 ? gl_FragCoord.z : log2( vFragDepth ) * logDepthBufFC * 0.5;
#endif`,Ng=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	uniform float logDepthBufFC;
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,Ug=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,Og=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	vFragDepth = 1.0 + gl_Position.w;
	vIsPerspective = float( isPerspectiveMatrix( projectionMatrix ) );
#endif`,Fg=`#ifdef USE_MAP
	vec4 sampledDiffuseColor = texture2D( map, vMapUv );
	#ifdef DECODE_VIDEO_TEXTURE
		sampledDiffuseColor = sRGBTransferEOTF( sampledDiffuseColor );
	#endif
	diffuseColor *= sampledDiffuseColor;
#endif`,Bg=`#ifdef USE_MAP
	uniform sampler2D map;
#endif`,kg=`#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
	#if defined( USE_POINTS_UV )
		vec2 uv = vUv;
	#else
		vec2 uv = ( uvTransform * vec3( gl_PointCoord.x, 1.0 - gl_PointCoord.y, 1 ) ).xy;
	#endif
#endif
#ifdef USE_MAP
	diffuseColor *= texture2D( map, uv );
#endif
#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, uv ).g;
#endif`,zg=`#if defined( USE_POINTS_UV )
	varying vec2 vUv;
#else
	#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
		uniform mat3 uvTransform;
	#endif
#endif
#ifdef USE_MAP
	uniform sampler2D map;
#endif
#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,Hg=`float metalnessFactor = metalness;
#ifdef USE_METALNESSMAP
	vec4 texelMetalness = texture2D( metalnessMap, vMetalnessMapUv );
	metalnessFactor *= texelMetalness.b;
#endif`,Vg=`#ifdef USE_METALNESSMAP
	uniform sampler2D metalnessMap;
#endif`,Gg=`#ifdef USE_INSTANCING_MORPH
	float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	float morphTargetBaseInfluence = texelFetch( morphTexture, ivec2( 0, gl_InstanceID ), 0 ).r;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		morphTargetInfluences[i] =  texelFetch( morphTexture, ivec2( i + 1, gl_InstanceID ), 0 ).r;
	}
#endif`,Wg=`#if defined( USE_MORPHCOLORS )
	vColor *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		#if defined( USE_COLOR_ALPHA )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ) * morphTargetInfluences[ i ];
		#elif defined( USE_COLOR )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ).rgb * morphTargetInfluences[ i ];
		#endif
	}
#endif`,Xg=`#ifdef USE_MORPHNORMALS
	objectNormal *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) objectNormal += getMorph( gl_VertexID, i, 1 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,qg=`#ifdef USE_MORPHTARGETS
	#ifndef USE_INSTANCING_MORPH
		uniform float morphTargetBaseInfluence;
		uniform float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	#endif
	uniform sampler2DArray morphTargetsTexture;
	uniform ivec2 morphTargetsTextureSize;
	vec4 getMorph( const in int vertexIndex, const in int morphTargetIndex, const in int offset ) {
		int texelIndex = vertexIndex * MORPHTARGETS_TEXTURE_STRIDE + offset;
		int y = texelIndex / morphTargetsTextureSize.x;
		int x = texelIndex - y * morphTargetsTextureSize.x;
		ivec3 morphUV = ivec3( x, y, morphTargetIndex );
		return texelFetch( morphTargetsTexture, morphUV, 0 );
	}
#endif`,Yg=`#ifdef USE_MORPHTARGETS
	transformed *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) transformed += getMorph( gl_VertexID, i, 0 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,Zg=`float faceDirection = gl_FrontFacing ? 1.0 : - 1.0;
#ifdef FLAT_SHADED
	vec3 fdx = dFdx( vViewPosition );
	vec3 fdy = dFdy( vViewPosition );
	vec3 normal = normalize( cross( fdx, fdy ) );
#else
	vec3 normal = normalize( vNormal );
	#ifdef DOUBLE_SIDED
		normal *= faceDirection;
	#endif
#endif
#if defined( USE_NORMALMAP_TANGENTSPACE ) || defined( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY )
	#ifdef USE_TANGENT
		mat3 tbn = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn = getTangentFrame( - vViewPosition, normal,
		#if defined( USE_NORMALMAP )
			vNormalMapUv
		#elif defined( USE_CLEARCOAT_NORMALMAP )
			vClearcoatNormalMapUv
		#else
			vUv
		#endif
		);
	#endif
	#ifdef DOUBLE_SIDED
		tbn[0] *= faceDirection;
		tbn[1] *= faceDirection;
	#endif
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	#ifdef USE_TANGENT
		mat3 tbn2 = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn2 = getTangentFrame( - vViewPosition, normal, vClearcoatNormalMapUv );
	#endif
	#ifdef DOUBLE_SIDED
		tbn2[0] *= faceDirection;
		tbn2[1] *= faceDirection;
	#endif
#endif
vec3 nonPerturbedNormal = normal;`,Kg=`#ifdef USE_NORMALMAP_OBJECTSPACE
	normal = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#ifdef FLIP_SIDED
		normal = - normal;
	#endif
	#ifdef DOUBLE_SIDED
		normal = normal * faceDirection;
	#endif
	normal = normalize( normalMatrix * normal );
#elif defined( USE_NORMALMAP_TANGENTSPACE )
	vec3 mapN = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#if defined( USE_PACKED_NORMALMAP )
		mapN = vec3( mapN.xy, sqrt( saturate( 1.0 - dot( mapN.xy, mapN.xy ) ) ) );
	#endif
	mapN.xy *= normalScale;
	normal = normalize( tbn * mapN );
#elif defined( USE_BUMPMAP )
	normal = perturbNormalArb( - vViewPosition, normal, dHdxy_fwd(), faceDirection );
#endif`,jg=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,Jg=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,$g=`#ifndef FLAT_SHADED
	vNormal = normalize( transformedNormal );
	#ifdef USE_TANGENT
		vTangent = normalize( transformedTangent );
		vBitangent = normalize( cross( vNormal, vTangent ) * tangent.w );
		#ifdef FLIP_SIDED
			vBitangent = - vBitangent;
		#endif
	#endif
#endif`,Qg=`#ifdef USE_NORMALMAP
	uniform sampler2D normalMap;
	uniform vec2 normalScale;
#endif
#ifdef USE_NORMALMAP_OBJECTSPACE
	uniform mat3 normalMatrix;
#endif
#if ! defined ( USE_TANGENT ) && ( defined ( USE_NORMALMAP_TANGENTSPACE ) || defined ( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY ) )
	mat3 getTangentFrame( vec3 eye_pos, vec3 surf_norm, vec2 uv ) {
		vec3 q0 = dFdx( eye_pos.xyz );
		vec3 q1 = dFdy( eye_pos.xyz );
		vec2 st0 = dFdx( uv.st );
		vec2 st1 = dFdy( uv.st );
		vec3 N = surf_norm;
		vec3 q1perp = cross( q1, N );
		vec3 q0perp = cross( N, q0 );
		vec3 T = q1perp * st0.x + q0perp * st1.x;
		vec3 B = q1perp * st0.y + q0perp * st1.y;
		float det = max( dot( T, T ), dot( B, B ) );
		float scale = ( det == 0.0 ) ? 0.0 : inversesqrt( det );
		return mat3( T * scale, B * scale, N );
	}
#endif`,e0=`#ifdef USE_CLEARCOAT
	vec3 clearcoatNormal = nonPerturbedNormal;
#endif`,t0=`#ifdef USE_CLEARCOAT_NORMALMAP
	vec3 clearcoatMapN = texture2D( clearcoatNormalMap, vClearcoatNormalMapUv ).xyz * 2.0 - 1.0;
	clearcoatMapN.xy *= clearcoatNormalScale;
	clearcoatNormal = normalize( tbn2 * clearcoatMapN );
#endif`,n0=`#ifdef USE_CLEARCOATMAP
	uniform sampler2D clearcoatMap;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform sampler2D clearcoatNormalMap;
	uniform vec2 clearcoatNormalScale;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform sampler2D clearcoatRoughnessMap;
#endif`,i0=`#ifdef USE_IRIDESCENCEMAP
	uniform sampler2D iridescenceMap;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform sampler2D iridescenceThicknessMap;
#endif`,s0=`#ifdef OPAQUE
diffuseColor.a = 1.0;
#endif
#ifdef USE_TRANSMISSION
diffuseColor.a *= material.transmissionAlpha;
#endif
gl_FragColor = vec4( outgoingLight, diffuseColor.a );`,r0=`vec3 packNormalToRGB( const in vec3 normal ) {
	return normalize( normal ) * 0.5 + 0.5;
}
vec3 unpackRGBToNormal( const in vec3 rgb ) {
	return 2.0 * rgb.xyz - 1.0;
}
const float PackUpscale = 256. / 255.;const float UnpackDownscale = 255. / 256.;const float ShiftRight8 = 1. / 256.;
const float Inv255 = 1. / 255.;
const vec4 PackFactors = vec4( 1.0, 256.0, 256.0 * 256.0, 256.0 * 256.0 * 256.0 );
const vec2 UnpackFactors2 = vec2( UnpackDownscale, 1.0 / PackFactors.g );
const vec3 UnpackFactors3 = vec3( UnpackDownscale / PackFactors.rg, 1.0 / PackFactors.b );
const vec4 UnpackFactors4 = vec4( UnpackDownscale / PackFactors.rgb, 1.0 / PackFactors.a );
vec4 packDepthToRGBA( const in float v ) {
	if( v <= 0.0 )
		return vec4( 0., 0., 0., 0. );
	if( v >= 1.0 )
		return vec4( 1., 1., 1., 1. );
	float vuf;
	float af = modf( v * PackFactors.a, vuf );
	float bf = modf( vuf * ShiftRight8, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec4( vuf * Inv255, gf * PackUpscale, bf * PackUpscale, af );
}
vec3 packDepthToRGB( const in float v ) {
	if( v <= 0.0 )
		return vec3( 0., 0., 0. );
	if( v >= 1.0 )
		return vec3( 1., 1., 1. );
	float vuf;
	float bf = modf( v * PackFactors.b, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec3( vuf * Inv255, gf * PackUpscale, bf );
}
vec2 packDepthToRG( const in float v ) {
	if( v <= 0.0 )
		return vec2( 0., 0. );
	if( v >= 1.0 )
		return vec2( 1., 1. );
	float vuf;
	float gf = modf( v * 256., vuf );
	return vec2( vuf * Inv255, gf );
}
float unpackRGBAToDepth( const in vec4 v ) {
	return dot( v, UnpackFactors4 );
}
float unpackRGBToDepth( const in vec3 v ) {
	return dot( v, UnpackFactors3 );
}
float unpackRGToDepth( const in vec2 v ) {
	return v.r * UnpackFactors2.r + v.g * UnpackFactors2.g;
}
vec4 pack2HalfToRGBA( const in vec2 v ) {
	vec4 r = vec4( v.x, fract( v.x * 255.0 ), v.y, fract( v.y * 255.0 ) );
	return vec4( r.x - r.y / 255.0, r.y, r.z - r.w / 255.0, r.w );
}
vec2 unpackRGBATo2Half( const in vec4 v ) {
	return vec2( v.x + ( v.y / 255.0 ), v.z + ( v.w / 255.0 ) );
}
float viewZToOrthographicDepth( const in float viewZ, const in float near, const in float far ) {
	return ( viewZ + near ) / ( near - far );
}
float orthographicDepthToViewZ( const in float depth, const in float near, const in float far ) {
	#ifdef USE_REVERSED_DEPTH_BUFFER
	
		return depth * ( far - near ) - far;
	#else
		return depth * ( near - far ) - near;
	#endif
}
float viewZToPerspectiveDepth( const in float viewZ, const in float near, const in float far ) {
	return ( ( near + viewZ ) * far ) / ( ( far - near ) * viewZ );
}
float perspectiveDepthToViewZ( const in float depth, const in float near, const in float far ) {
	
	#ifdef USE_REVERSED_DEPTH_BUFFER
		return ( near * far ) / ( ( near - far ) * depth - near );
	#else
		return ( near * far ) / ( ( far - near ) * depth - far );
	#endif
}`,o0=`#ifdef PREMULTIPLIED_ALPHA
	gl_FragColor.rgb *= gl_FragColor.a;
#endif`,a0=`vec4 mvPosition = vec4( transformed, 1.0 );
#ifdef USE_BATCHING
	mvPosition = batchingMatrix * mvPosition;
#endif
#ifdef USE_INSTANCING
	mvPosition = instanceMatrix * mvPosition;
#endif
mvPosition = modelViewMatrix * mvPosition;
gl_Position = projectionMatrix * mvPosition;`,c0=`#ifdef DITHERING
	gl_FragColor.rgb = dithering( gl_FragColor.rgb );
#endif`,l0=`#ifdef DITHERING
	vec3 dithering( vec3 color ) {
		float grid_position = rand( gl_FragCoord.xy );
		vec3 dither_shift_RGB = vec3( 0.25 / 255.0, -0.25 / 255.0, 0.25 / 255.0 );
		dither_shift_RGB = mix( 2.0 * dither_shift_RGB, -2.0 * dither_shift_RGB, grid_position );
		return color + dither_shift_RGB;
	}
#endif`,h0=`float roughnessFactor = roughness;
#ifdef USE_ROUGHNESSMAP
	vec4 texelRoughness = texture2D( roughnessMap, vRoughnessMapUv );
	roughnessFactor *= texelRoughness.g;
#endif`,u0=`#ifdef USE_ROUGHNESSMAP
	uniform sampler2D roughnessMap;
#endif`,f0=`#if NUM_SPOT_LIGHT_COORDS > 0
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#if NUM_SPOT_LIGHT_MAPS > 0
	uniform sampler2D spotLightMap[ NUM_SPOT_LIGHT_MAPS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#else
			uniform sampler2D directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#endif
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#else
			uniform sampler2D spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#endif
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform samplerCubeShadow pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#elif defined( SHADOWMAP_TYPE_BASIC )
			uniform samplerCube pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#endif
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float interleavedGradientNoise( vec2 position ) {
			return fract( 52.9829189 * fract( dot( position, vec2( 0.06711056, 0.00583715 ) ) ) );
		}
		vec2 vogelDiskSample( int sampleIndex, int samplesCount, float phi ) {
			const float goldenAngle = 2.399963229728653;
			float r = sqrt( ( float( sampleIndex ) + 0.5 ) / float( samplesCount ) );
			float theta = float( sampleIndex ) * goldenAngle + phi;
			return vec2( cos( theta ), sin( theta ) ) * r;
		}
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float getShadow( sampler2DShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			shadowCoord.z += shadowBias;
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 texelSize = vec2( 1.0 ) / shadowMapSize;
				float radius = shadowRadius * texelSize.x;
				float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
				shadow = (
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 0, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 1, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 2, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 3, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 4, 5, phi ) * radius, shadowCoord.z ) )
				) * 0.2;
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#elif defined( SHADOWMAP_TYPE_VSM )
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 distribution = texture2D( shadowMap, shadowCoord.xy ).rg;
				float mean = distribution.x;
				float variance = distribution.y * distribution.y;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					float hard_shadow = step( mean, shadowCoord.z );
				#else
					float hard_shadow = step( shadowCoord.z, mean );
				#endif
				
				if ( hard_shadow == 1.0 ) {
					shadow = 1.0;
				} else {
					variance = max( variance, 0.0000001 );
					float d = shadowCoord.z - mean;
					float p_max = variance / ( variance + d * d );
					p_max = clamp( ( p_max - 0.3 ) / 0.65, 0.0, 1.0 );
					shadow = max( hard_shadow, p_max );
				}
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#else
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				float depth = texture2D( shadowMap, shadowCoord.xy ).r;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					shadow = step( depth, shadowCoord.z );
				#else
					shadow = step( shadowCoord.z, depth );
				#endif
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
	#if defined( SHADOWMAP_TYPE_PCF )
	float getPointShadow( samplerCubeShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 bd3D = normalize( lightToPosition );
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			#ifdef USE_REVERSED_DEPTH_BUFFER
				float dp = ( shadowCameraNear * ( shadowCameraFar - viewSpaceZ ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp -= shadowBias;
			#else
				float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp += shadowBias;
			#endif
			float texelSize = shadowRadius / shadowMapSize.x;
			vec3 absDir = abs( bd3D );
			vec3 tangent = absDir.x > absDir.z ? vec3( 0.0, 1.0, 0.0 ) : vec3( 1.0, 0.0, 0.0 );
			tangent = normalize( cross( bd3D, tangent ) );
			vec3 bitangent = cross( bd3D, tangent );
			float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
			vec2 sample0 = vogelDiskSample( 0, 5, phi );
			vec2 sample1 = vogelDiskSample( 1, 5, phi );
			vec2 sample2 = vogelDiskSample( 2, 5, phi );
			vec2 sample3 = vogelDiskSample( 3, 5, phi );
			vec2 sample4 = vogelDiskSample( 4, 5, phi );
			shadow = (
				texture( shadowMap, vec4( bd3D + ( tangent * sample0.x + bitangent * sample0.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample1.x + bitangent * sample1.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample2.x + bitangent * sample2.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample3.x + bitangent * sample3.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample4.x + bitangent * sample4.y ) * texelSize, dp ) )
			) * 0.2;
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#elif defined( SHADOWMAP_TYPE_BASIC )
	float getPointShadow( samplerCube shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
			dp += shadowBias;
			vec3 bd3D = normalize( lightToPosition );
			float depth = textureCube( shadowMap, bd3D ).r;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				depth = 1.0 - depth;
			#endif
			shadow = step( dp, depth );
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#endif
	#endif
#endif`,d0=`#if NUM_SPOT_LIGHT_COORDS > 0
	uniform mat4 spotLightMatrix[ NUM_SPOT_LIGHT_COORDS ];
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_DIR_LIGHT_SHADOWS > 0
		uniform mat4 directionalShadowMatrix[ NUM_DIR_LIGHT_SHADOWS ];
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		uniform mat4 pointShadowMatrix[ NUM_POINT_LIGHT_SHADOWS ];
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
#endif`,p0=`#if ( defined( USE_SHADOWMAP ) && ( NUM_DIR_LIGHT_SHADOWS > 0 || NUM_POINT_LIGHT_SHADOWS > 0 ) ) || ( NUM_SPOT_LIGHT_COORDS > 0 )
	#ifdef HAS_NORMAL
		vec3 shadowWorldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
	#else
		vec3 shadowWorldNormal = vec3( 0.0 );
	#endif
	vec4 shadowWorldPosition;
#endif
#if defined( USE_SHADOWMAP )
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * directionalLightShadows[ i ].shadowNormalBias, 0 );
			vDirectionalShadowCoord[ i ] = directionalShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * pointLightShadows[ i ].shadowNormalBias, 0 );
			vPointShadowCoord[ i ] = pointShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
#endif
#if NUM_SPOT_LIGHT_COORDS > 0
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_COORDS; i ++ ) {
		shadowWorldPosition = worldPosition;
		#if ( defined( USE_SHADOWMAP ) && UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
			shadowWorldPosition.xyz += shadowWorldNormal * spotLightShadows[ i ].shadowNormalBias;
		#endif
		vSpotLightCoord[ i ] = spotLightMatrix[ i ] * shadowWorldPosition;
	}
	#pragma unroll_loop_end
#endif`,m0=`float getShadowMask() {
	float shadow = 1.0;
	#ifdef USE_SHADOWMAP
	#if NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
		directionalLight = directionalLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( directionalShadowMap[ i ], directionalLight.shadowMapSize, directionalLight.shadowIntensity, directionalLight.shadowBias, directionalLight.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_SHADOWS; i ++ ) {
		spotLight = spotLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( spotShadowMap[ i ], spotLight.shadowMapSize, spotLight.shadowIntensity, spotLight.shadowBias, spotLight.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0 && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
	PointLightShadow pointLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
		pointLight = pointLightShadows[ i ];
		shadow *= receiveShadow ? getPointShadow( pointShadowMap[ i ], pointLight.shadowMapSize, pointLight.shadowIntensity, pointLight.shadowBias, pointLight.shadowRadius, vPointShadowCoord[ i ], pointLight.shadowCameraNear, pointLight.shadowCameraFar ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#endif
	return shadow;
}`,g0=`#ifdef USE_SKINNING
	mat4 boneMatX = getBoneMatrix( skinIndex.x );
	mat4 boneMatY = getBoneMatrix( skinIndex.y );
	mat4 boneMatZ = getBoneMatrix( skinIndex.z );
	mat4 boneMatW = getBoneMatrix( skinIndex.w );
#endif`,_0=`#ifdef USE_SKINNING
	uniform mat4 bindMatrix;
	uniform mat4 bindMatrixInverse;
	uniform highp sampler2D boneTexture;
	mat4 getBoneMatrix( const in float i ) {
		int size = textureSize( boneTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( boneTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( boneTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( boneTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( boneTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
#endif`,x0=`#ifdef USE_SKINNING
	vec4 skinVertex = bindMatrix * vec4( transformed, 1.0 );
	vec4 skinned = vec4( 0.0 );
	skinned += boneMatX * skinVertex * skinWeight.x;
	skinned += boneMatY * skinVertex * skinWeight.y;
	skinned += boneMatZ * skinVertex * skinWeight.z;
	skinned += boneMatW * skinVertex * skinWeight.w;
	transformed = ( bindMatrixInverse * skinned ).xyz;
#endif`,y0=`#ifdef USE_SKINNING
	mat4 skinMatrix = mat4( 0.0 );
	skinMatrix += skinWeight.x * boneMatX;
	skinMatrix += skinWeight.y * boneMatY;
	skinMatrix += skinWeight.z * boneMatZ;
	skinMatrix += skinWeight.w * boneMatW;
	skinMatrix = bindMatrixInverse * skinMatrix * bindMatrix;
	objectNormal = vec4( skinMatrix * vec4( objectNormal, 0.0 ) ).xyz;
	#ifdef USE_TANGENT
		objectTangent = vec4( skinMatrix * vec4( objectTangent, 0.0 ) ).xyz;
	#endif
#endif`,v0=`float specularStrength;
#ifdef USE_SPECULARMAP
	vec4 texelSpecular = texture2D( specularMap, vSpecularMapUv );
	specularStrength = texelSpecular.r;
#else
	specularStrength = 1.0;
#endif`,b0=`#ifdef USE_SPECULARMAP
	uniform sampler2D specularMap;
#endif`,M0=`#if defined( TONE_MAPPING )
	gl_FragColor.rgb = toneMapping( gl_FragColor.rgb );
#endif`,S0=`#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
uniform float toneMappingExposure;
vec3 LinearToneMapping( vec3 color ) {
	return saturate( toneMappingExposure * color );
}
vec3 ReinhardToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	return saturate( color / ( vec3( 1.0 ) + color ) );
}
vec3 CineonToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	color = max( vec3( 0.0 ), color - 0.004 );
	return pow( ( color * ( 6.2 * color + 0.5 ) ) / ( color * ( 6.2 * color + 1.7 ) + 0.06 ), vec3( 2.2 ) );
}
vec3 RRTAndODTFit( vec3 v ) {
	vec3 a = v * ( v + 0.0245786 ) - 0.000090537;
	vec3 b = v * ( 0.983729 * v + 0.4329510 ) + 0.238081;
	return a / b;
}
vec3 ACESFilmicToneMapping( vec3 color ) {
	const mat3 ACESInputMat = mat3(
		vec3( 0.59719, 0.07600, 0.02840 ),		vec3( 0.35458, 0.90834, 0.13383 ),
		vec3( 0.04823, 0.01566, 0.83777 )
	);
	const mat3 ACESOutputMat = mat3(
		vec3(  1.60475, -0.10208, -0.00327 ),		vec3( -0.53108,  1.10813, -0.07276 ),
		vec3( -0.07367, -0.00605,  1.07602 )
	);
	color *= toneMappingExposure / 0.6;
	color = ACESInputMat * color;
	color = RRTAndODTFit( color );
	color = ACESOutputMat * color;
	return saturate( color );
}
const mat3 LINEAR_REC2020_TO_LINEAR_SRGB = mat3(
	vec3( 1.6605, - 0.1246, - 0.0182 ),
	vec3( - 0.5876, 1.1329, - 0.1006 ),
	vec3( - 0.0728, - 0.0083, 1.1187 )
);
const mat3 LINEAR_SRGB_TO_LINEAR_REC2020 = mat3(
	vec3( 0.6274, 0.0691, 0.0164 ),
	vec3( 0.3293, 0.9195, 0.0880 ),
	vec3( 0.0433, 0.0113, 0.8956 )
);
vec3 agxDefaultContrastApprox( vec3 x ) {
	vec3 x2 = x * x;
	vec3 x4 = x2 * x2;
	return + 15.5 * x4 * x2
		- 40.14 * x4 * x
		+ 31.96 * x4
		- 6.868 * x2 * x
		+ 0.4298 * x2
		+ 0.1191 * x
		- 0.00232;
}
vec3 AgXToneMapping( vec3 color ) {
	const mat3 AgXInsetMatrix = mat3(
		vec3( 0.856627153315983, 0.137318972929847, 0.11189821299995 ),
		vec3( 0.0951212405381588, 0.761241990602591, 0.0767994186031903 ),
		vec3( 0.0482516061458583, 0.101439036467562, 0.811302368396859 )
	);
	const mat3 AgXOutsetMatrix = mat3(
		vec3( 1.1271005818144368, - 0.1413297634984383, - 0.14132976349843826 ),
		vec3( - 0.11060664309660323, 1.157823702216272, - 0.11060664309660294 ),
		vec3( - 0.016493938717834573, - 0.016493938717834257, 1.2519364065950405 )
	);
	const float AgxMinEv = - 12.47393;	const float AgxMaxEv = 4.026069;
	color *= toneMappingExposure;
	color = LINEAR_SRGB_TO_LINEAR_REC2020 * color;
	color = AgXInsetMatrix * color;
	color = max( color, 1e-10 );	color = log2( color );
	color = ( color - AgxMinEv ) / ( AgxMaxEv - AgxMinEv );
	color = clamp( color, 0.0, 1.0 );
	color = agxDefaultContrastApprox( color );
	color = AgXOutsetMatrix * color;
	color = pow( max( vec3( 0.0 ), color ), vec3( 2.2 ) );
	color = LINEAR_REC2020_TO_LINEAR_SRGB * color;
	color = clamp( color, 0.0, 1.0 );
	return color;
}
vec3 NeutralToneMapping( vec3 color ) {
	const float StartCompression = 0.8 - 0.04;
	const float Desaturation = 0.15;
	color *= toneMappingExposure;
	float x = min( color.r, min( color.g, color.b ) );
	float offset = x < 0.08 ? x - 6.25 * x * x : 0.04;
	color -= offset;
	float peak = max( color.r, max( color.g, color.b ) );
	if ( peak < StartCompression ) return color;
	float d = 1. - StartCompression;
	float newPeak = 1. - d * d / ( peak + d - StartCompression );
	color *= newPeak / peak;
	float g = 1. - 1. / ( Desaturation * ( peak - newPeak ) + 1. );
	return mix( color, vec3( newPeak ), g );
}
vec3 CustomToneMapping( vec3 color ) { return color; }`,T0=`#ifdef USE_TRANSMISSION
	material.transmission = transmission;
	material.transmissionAlpha = 1.0;
	material.thickness = thickness;
	material.attenuationDistance = attenuationDistance;
	material.attenuationColor = attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		material.transmission *= texture2D( transmissionMap, vTransmissionMapUv ).r;
	#endif
	#ifdef USE_THICKNESSMAP
		material.thickness *= texture2D( thicknessMap, vThicknessMapUv ).g;
	#endif
	vec3 pos = vWorldPosition;
	vec3 v = normalize( cameraPosition - pos );
	vec3 n = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec4 transmitted = getIBLVolumeRefraction(
		n, v, material.roughness, material.diffuseContribution, material.specularColorBlended, material.specularF90,
		pos, modelMatrix, viewMatrix, projectionMatrix, material.dispersion, material.ior, material.thickness,
		material.attenuationColor, material.attenuationDistance );
	material.transmissionAlpha = mix( material.transmissionAlpha, transmitted.a, material.transmission );
	totalDiffuse = mix( totalDiffuse, transmitted.rgb, material.transmission );
#endif`,E0=`#ifdef USE_TRANSMISSION
	uniform float transmission;
	uniform float thickness;
	uniform float attenuationDistance;
	uniform vec3 attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		uniform sampler2D transmissionMap;
	#endif
	#ifdef USE_THICKNESSMAP
		uniform sampler2D thicknessMap;
	#endif
	uniform vec2 transmissionSamplerSize;
	uniform sampler2D transmissionSamplerMap;
	uniform mat4 modelMatrix;
	uniform mat4 projectionMatrix;
	varying vec3 vWorldPosition;
	float w0( float a ) {
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - a + 3.0 ) - 3.0 ) + 1.0 );
	}
	float w1( float a ) {
		return ( 1.0 / 6.0 ) * ( a *  a * ( 3.0 * a - 6.0 ) + 4.0 );
	}
	float w2( float a ){
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - 3.0 * a + 3.0 ) + 3.0 ) + 1.0 );
	}
	float w3( float a ) {
		return ( 1.0 / 6.0 ) * ( a * a * a );
	}
	float g0( float a ) {
		return w0( a ) + w1( a );
	}
	float g1( float a ) {
		return w2( a ) + w3( a );
	}
	float h0( float a ) {
		return - 1.0 + w1( a ) / ( w0( a ) + w1( a ) );
	}
	float h1( float a ) {
		return 1.0 + w3( a ) / ( w2( a ) + w3( a ) );
	}
	vec4 bicubic( sampler2D tex, vec2 uv, vec4 texelSize, float lod ) {
		uv = uv * texelSize.zw + 0.5;
		vec2 iuv = floor( uv );
		vec2 fuv = fract( uv );
		float g0x = g0( fuv.x );
		float g1x = g1( fuv.x );
		float h0x = h0( fuv.x );
		float h1x = h1( fuv.x );
		float h0y = h0( fuv.y );
		float h1y = h1( fuv.y );
		vec2 p0 = ( vec2( iuv.x + h0x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p1 = ( vec2( iuv.x + h1x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p2 = ( vec2( iuv.x + h0x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		vec2 p3 = ( vec2( iuv.x + h1x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		return g0( fuv.y ) * ( g0x * textureLod( tex, p0, lod ) + g1x * textureLod( tex, p1, lod ) ) +
			g1( fuv.y ) * ( g0x * textureLod( tex, p2, lod ) + g1x * textureLod( tex, p3, lod ) );
	}
	vec4 textureBicubic( sampler2D sampler, vec2 uv, float lod ) {
		vec2 fLodSize = vec2( textureSize( sampler, int( lod ) ) );
		vec2 cLodSize = vec2( textureSize( sampler, int( lod + 1.0 ) ) );
		vec2 fLodSizeInv = 1.0 / fLodSize;
		vec2 cLodSizeInv = 1.0 / cLodSize;
		vec4 fSample = bicubic( sampler, uv, vec4( fLodSizeInv, fLodSize ), floor( lod ) );
		vec4 cSample = bicubic( sampler, uv, vec4( cLodSizeInv, cLodSize ), ceil( lod ) );
		return mix( fSample, cSample, fract( lod ) );
	}
	vec3 getVolumeTransmissionRay( const in vec3 n, const in vec3 v, const in float thickness, const in float ior, const in mat4 modelMatrix ) {
		vec3 refractionVector = refract( - v, normalize( n ), 1.0 / ior );
		vec3 modelScale;
		modelScale.x = length( vec3( modelMatrix[ 0 ].xyz ) );
		modelScale.y = length( vec3( modelMatrix[ 1 ].xyz ) );
		modelScale.z = length( vec3( modelMatrix[ 2 ].xyz ) );
		return normalize( refractionVector ) * thickness * modelScale;
	}
	float applyIorToRoughness( const in float roughness, const in float ior ) {
		return roughness * clamp( ior * 2.0 - 2.0, 0.0, 1.0 );
	}
	vec4 getTransmissionSample( const in vec2 fragCoord, const in float roughness, const in float ior ) {
		float lod = log2( transmissionSamplerSize.x ) * applyIorToRoughness( roughness, ior );
		return textureBicubic( transmissionSamplerMap, fragCoord.xy, lod );
	}
	vec3 volumeAttenuation( const in float transmissionDistance, const in vec3 attenuationColor, const in float attenuationDistance ) {
		if ( isinf( attenuationDistance ) ) {
			return vec3( 1.0 );
		} else {
			vec3 attenuationCoefficient = -log( attenuationColor ) / attenuationDistance;
			vec3 transmittance = exp( - attenuationCoefficient * transmissionDistance );			return transmittance;
		}
	}
	vec4 getIBLVolumeRefraction( const in vec3 n, const in vec3 v, const in float roughness, const in vec3 diffuseColor,
		const in vec3 specularColor, const in float specularF90, const in vec3 position, const in mat4 modelMatrix,
		const in mat4 viewMatrix, const in mat4 projMatrix, const in float dispersion, const in float ior, const in float thickness,
		const in vec3 attenuationColor, const in float attenuationDistance ) {
		vec4 transmittedLight;
		vec3 transmittance;
		#ifdef USE_DISPERSION
			float halfSpread = ( ior - 1.0 ) * 0.025 * dispersion;
			vec3 iors = vec3( ior - halfSpread, ior, ior + halfSpread );
			for ( int i = 0; i < 3; i ++ ) {
				vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, iors[ i ], modelMatrix );
				vec3 refractedRayExit = position + transmissionRay;
				vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
				vec2 refractionCoords = ndcPos.xy / ndcPos.w;
				refractionCoords += 1.0;
				refractionCoords /= 2.0;
				vec4 transmissionSample = getTransmissionSample( refractionCoords, roughness, iors[ i ] );
				transmittedLight[ i ] = transmissionSample[ i ];
				transmittedLight.a += transmissionSample.a;
				transmittance[ i ] = diffuseColor[ i ] * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance )[ i ];
			}
			transmittedLight.a /= 3.0;
		#else
			vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, ior, modelMatrix );
			vec3 refractedRayExit = position + transmissionRay;
			vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
			vec2 refractionCoords = ndcPos.xy / ndcPos.w;
			refractionCoords += 1.0;
			refractionCoords /= 2.0;
			transmittedLight = getTransmissionSample( refractionCoords, roughness, ior );
			transmittance = diffuseColor * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance );
		#endif
		vec3 attenuatedColor = transmittance * transmittedLight.rgb;
		vec3 F = EnvironmentBRDF( n, v, specularColor, specularF90, roughness );
		float transmittanceFactor = ( transmittance.r + transmittance.g + transmittance.b ) / 3.0;
		return vec4( ( 1.0 - F ) * attenuatedColor, 1.0 - ( 1.0 - transmittedLight.a ) * transmittanceFactor );
	}
#endif`,A0=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_SPECULARMAP
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,w0=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	uniform mat3 mapTransform;
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	uniform mat3 alphaMapTransform;
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	uniform mat3 lightMapTransform;
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	uniform mat3 aoMapTransform;
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	uniform mat3 bumpMapTransform;
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	uniform mat3 normalMapTransform;
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_DISPLACEMENTMAP
	uniform mat3 displacementMapTransform;
	varying vec2 vDisplacementMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	uniform mat3 emissiveMapTransform;
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	uniform mat3 metalnessMapTransform;
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	uniform mat3 roughnessMapTransform;
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	uniform mat3 anisotropyMapTransform;
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	uniform mat3 clearcoatMapTransform;
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform mat3 clearcoatNormalMapTransform;
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform mat3 clearcoatRoughnessMapTransform;
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	uniform mat3 sheenColorMapTransform;
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	uniform mat3 sheenRoughnessMapTransform;
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	uniform mat3 iridescenceMapTransform;
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform mat3 iridescenceThicknessMapTransform;
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SPECULARMAP
	uniform mat3 specularMapTransform;
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	uniform mat3 specularColorMapTransform;
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	uniform mat3 specularIntensityMapTransform;
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,R0=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	vUv = vec3( uv, 1 ).xy;
#endif
#ifdef USE_MAP
	vMapUv = ( mapTransform * vec3( MAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ALPHAMAP
	vAlphaMapUv = ( alphaMapTransform * vec3( ALPHAMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_LIGHTMAP
	vLightMapUv = ( lightMapTransform * vec3( LIGHTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_AOMAP
	vAoMapUv = ( aoMapTransform * vec3( AOMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_BUMPMAP
	vBumpMapUv = ( bumpMapTransform * vec3( BUMPMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_NORMALMAP
	vNormalMapUv = ( normalMapTransform * vec3( NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_DISPLACEMENTMAP
	vDisplacementMapUv = ( displacementMapTransform * vec3( DISPLACEMENTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_EMISSIVEMAP
	vEmissiveMapUv = ( emissiveMapTransform * vec3( EMISSIVEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_METALNESSMAP
	vMetalnessMapUv = ( metalnessMapTransform * vec3( METALNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ROUGHNESSMAP
	vRoughnessMapUv = ( roughnessMapTransform * vec3( ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ANISOTROPYMAP
	vAnisotropyMapUv = ( anisotropyMapTransform * vec3( ANISOTROPYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOATMAP
	vClearcoatMapUv = ( clearcoatMapTransform * vec3( CLEARCOATMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	vClearcoatNormalMapUv = ( clearcoatNormalMapTransform * vec3( CLEARCOAT_NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	vClearcoatRoughnessMapUv = ( clearcoatRoughnessMapTransform * vec3( CLEARCOAT_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCEMAP
	vIridescenceMapUv = ( iridescenceMapTransform * vec3( IRIDESCENCEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	vIridescenceThicknessMapUv = ( iridescenceThicknessMapTransform * vec3( IRIDESCENCE_THICKNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_COLORMAP
	vSheenColorMapUv = ( sheenColorMapTransform * vec3( SHEEN_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	vSheenRoughnessMapUv = ( sheenRoughnessMapTransform * vec3( SHEEN_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULARMAP
	vSpecularMapUv = ( specularMapTransform * vec3( SPECULARMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_COLORMAP
	vSpecularColorMapUv = ( specularColorMapTransform * vec3( SPECULAR_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	vSpecularIntensityMapUv = ( specularIntensityMapTransform * vec3( SPECULAR_INTENSITYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_TRANSMISSIONMAP
	vTransmissionMapUv = ( transmissionMapTransform * vec3( TRANSMISSIONMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_THICKNESSMAP
	vThicknessMapUv = ( thicknessMapTransform * vec3( THICKNESSMAP_UV, 1 ) ).xy;
#endif`,C0=`#if defined( USE_ENVMAP ) || defined( DISTANCE ) || defined ( USE_SHADOWMAP ) || defined ( USE_TRANSMISSION ) || NUM_SPOT_LIGHT_COORDS > 0
	vec4 worldPosition = vec4( transformed, 1.0 );
	#ifdef USE_BATCHING
		worldPosition = batchingMatrix * worldPosition;
	#endif
	#ifdef USE_INSTANCING
		worldPosition = instanceMatrix * worldPosition;
	#endif
	worldPosition = modelMatrix * worldPosition;
#endif`,P0=`varying vec2 vUv;
uniform mat3 uvTransform;
void main() {
	vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	gl_Position = vec4( position.xy, 1.0, 1.0 );
}`,I0=`uniform sampler2D t2D;
uniform float backgroundIntensity;
varying vec2 vUv;
void main() {
	vec4 texColor = texture2D( t2D, vUv );
	#ifdef DECODE_VIDEO_TEXTURE
		texColor = vec4( mix( pow( texColor.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), texColor.rgb * 0.0773993808, vec3( lessThanEqual( texColor.rgb, vec3( 0.04045 ) ) ) ), texColor.w );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,L0=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,D0=`#ifdef ENVMAP_TYPE_CUBE
	uniform samplerCube envMap;
#elif defined( ENVMAP_TYPE_CUBE_UV )
	uniform sampler2D envMap;
#endif
uniform float backgroundBlurriness;
uniform float backgroundIntensity;
uniform mat3 backgroundRotation;
varying vec3 vWorldDirection;
#include <cube_uv_reflection_fragment>
void main() {
	#ifdef ENVMAP_TYPE_CUBE
		vec4 texColor = textureCube( envMap, backgroundRotation * vWorldDirection );
	#elif defined( ENVMAP_TYPE_CUBE_UV )
		vec4 texColor = textureCubeUV( envMap, backgroundRotation * vWorldDirection, backgroundBlurriness );
	#else
		vec4 texColor = vec4( 0.0, 0.0, 0.0, 1.0 );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,N0=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,U0=`uniform samplerCube tCube;
uniform float tFlip;
uniform float opacity;
varying vec3 vWorldDirection;
void main() {
	vec4 texColor = textureCube( tCube, vec3( tFlip * vWorldDirection.x, vWorldDirection.yz ) );
	gl_FragColor = texColor;
	gl_FragColor.a *= opacity;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,O0=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
varying vec2 vHighPrecisionZW;
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vHighPrecisionZW = gl_Position.zw;
}`,F0=`#if DEPTH_PACKING == 3200
	uniform float opacity;
#endif
#include <common>
#include <packing>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
varying vec2 vHighPrecisionZW;
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#if DEPTH_PACKING == 3200
		diffuseColor.a = opacity;
	#endif
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <logdepthbuf_fragment>
	#ifdef USE_REVERSED_DEPTH_BUFFER
		float fragCoordZ = vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ];
	#else
		float fragCoordZ = 0.5 * vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ] + 0.5;
	#endif
	#if DEPTH_PACKING == 3200
		gl_FragColor = vec4( vec3( 1.0 - fragCoordZ ), opacity );
	#elif DEPTH_PACKING == 3201
		gl_FragColor = packDepthToRGBA( fragCoordZ );
	#elif DEPTH_PACKING == 3202
		gl_FragColor = vec4( packDepthToRGB( fragCoordZ ), 1.0 );
	#elif DEPTH_PACKING == 3203
		gl_FragColor = vec4( packDepthToRG( fragCoordZ ), 0.0, 1.0 );
	#endif
}`,B0=`#define DISTANCE
varying vec3 vWorldPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <worldpos_vertex>
	#include <clipping_planes_vertex>
	vWorldPosition = worldPosition.xyz;
}`,k0=`#define DISTANCE
uniform vec3 referencePosition;
uniform float nearDistance;
uniform float farDistance;
varying vec3 vWorldPosition;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	float dist = length( vWorldPosition - referencePosition );
	dist = ( dist - nearDistance ) / ( farDistance - nearDistance );
	dist = saturate( dist );
	gl_FragColor = vec4( dist, 0.0, 0.0, 1.0 );
}`,z0=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
}`,H0=`uniform sampler2D tEquirect;
varying vec3 vWorldDirection;
#include <common>
void main() {
	vec3 direction = normalize( vWorldDirection );
	vec2 sampleUV = equirectUv( direction );
	gl_FragColor = texture2D( tEquirect, sampleUV );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,V0=`uniform float scale;
attribute float lineDistance;
varying float vLineDistance;
#include <common>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	vLineDistance = scale * lineDistance;
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,G0=`uniform vec3 diffuse;
uniform float opacity;
uniform float dashSize;
uniform float totalSize;
varying float vLineDistance;
#include <common>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	if ( mod( vLineDistance, totalSize ) > dashSize ) {
		discard;
	}
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,W0=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#if defined ( USE_ENVMAP ) || defined ( USE_SKINNING )
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinbase_vertex>
		#include <skinnormal_vertex>
		#include <defaultnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <fog_vertex>
}`,X0=`uniform vec3 diffuse;
uniform float opacity;
#ifndef FLAT_SHADED
	varying vec3 vNormal;
#endif
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <fog_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		reflectedLight.indirectDiffuse += lightMapTexel.rgb * lightMapIntensity * RECIPROCAL_PI;
	#else
		reflectedLight.indirectDiffuse += vec3( 1.0 );
	#endif
	#include <aomap_fragment>
	reflectedLight.indirectDiffuse *= diffuseColor.rgb;
	vec3 outgoingLight = reflectedLight.indirectDiffuse;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,q0=`#define LAMBERT
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,Y0=`#define LAMBERT
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_lambert_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_lambert_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,Z0=`#define MATCAP
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <displacementmap_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
	vViewPosition = - mvPosition.xyz;
}`,K0=`#define MATCAP
uniform vec3 diffuse;
uniform float opacity;
uniform sampler2D matcap;
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	vec3 viewDir = normalize( vViewPosition );
	vec3 x = normalize( vec3( viewDir.z, 0.0, - viewDir.x ) );
	vec3 y = cross( viewDir, x );
	vec2 uv = vec2( dot( x, normal ), dot( y, normal ) ) * 0.495 + 0.5;
	#ifdef USE_MATCAP
		vec4 matcapColor = texture2D( matcap, uv );
	#else
		vec4 matcapColor = vec4( vec3( mix( 0.2, 0.8, uv.y ) ), 1.0 );
	#endif
	vec3 outgoingLight = diffuseColor.rgb * matcapColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,j0=`#define NORMAL
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	vViewPosition = - mvPosition.xyz;
#endif
}`,J0=`#define NORMAL
uniform float opacity;
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <uv_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 0.0, 0.0, 0.0, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	gl_FragColor = vec4( normalize( normal ) * 0.5 + 0.5, diffuseColor.a );
	#ifdef OPAQUE
		gl_FragColor.a = 1.0;
	#endif
}`,$0=`#define PHONG
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,Q0=`#define PHONG
uniform vec3 diffuse;
uniform vec3 emissive;
uniform vec3 specular;
uniform float shininess;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_phong_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_phong_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + reflectedLight.directSpecular + reflectedLight.indirectSpecular + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,e_=`#define STANDARD
varying vec3 vViewPosition;
#ifdef USE_TRANSMISSION
	varying vec3 vWorldPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
#ifdef USE_TRANSMISSION
	vWorldPosition = worldPosition.xyz;
#endif
}`,t_=`#define STANDARD
#ifdef PHYSICAL
	#define IOR
	#define USE_SPECULAR
#endif
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float roughness;
uniform float metalness;
uniform float opacity;
#ifdef IOR
	uniform float ior;
#endif
#ifdef USE_SPECULAR
	uniform float specularIntensity;
	uniform vec3 specularColor;
	#ifdef USE_SPECULAR_COLORMAP
		uniform sampler2D specularColorMap;
	#endif
	#ifdef USE_SPECULAR_INTENSITYMAP
		uniform sampler2D specularIntensityMap;
	#endif
#endif
#ifdef USE_CLEARCOAT
	uniform float clearcoat;
	uniform float clearcoatRoughness;
#endif
#ifdef USE_DISPERSION
	uniform float dispersion;
#endif
#ifdef USE_IRIDESCENCE
	uniform float iridescence;
	uniform float iridescenceIOR;
	uniform float iridescenceThicknessMinimum;
	uniform float iridescenceThicknessMaximum;
#endif
#ifdef USE_SHEEN
	uniform vec3 sheenColor;
	uniform float sheenRoughness;
	#ifdef USE_SHEEN_COLORMAP
		uniform sampler2D sheenColorMap;
	#endif
	#ifdef USE_SHEEN_ROUGHNESSMAP
		uniform sampler2D sheenRoughnessMap;
	#endif
#endif
#ifdef USE_ANISOTROPY
	uniform vec2 anisotropyVector;
	#ifdef USE_ANISOTROPYMAP
		uniform sampler2D anisotropyMap;
	#endif
#endif
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <iridescence_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_physical_pars_fragment>
#include <transmission_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <clearcoat_pars_fragment>
#include <iridescence_pars_fragment>
#include <roughnessmap_pars_fragment>
#include <metalnessmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <roughnessmap_fragment>
	#include <metalnessmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <clearcoat_normal_fragment_begin>
	#include <clearcoat_normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_physical_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 totalDiffuse = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse;
	vec3 totalSpecular = reflectedLight.directSpecular + reflectedLight.indirectSpecular;
	#include <transmission_fragment>
	vec3 outgoingLight = totalDiffuse + totalSpecular + totalEmissiveRadiance;
	#ifdef USE_SHEEN
 
		outgoingLight = outgoingLight + sheenSpecularDirect + sheenSpecularIndirect;
 
 	#endif
	#ifdef USE_CLEARCOAT
		float dotNVcc = saturate( dot( geometryClearcoatNormal, geometryViewDir ) );
		vec3 Fcc = F_Schlick( material.clearcoatF0, material.clearcoatF90, dotNVcc );
		outgoingLight = outgoingLight * ( 1.0 - material.clearcoat * Fcc ) + ( clearcoatSpecularDirect + clearcoatSpecularIndirect ) * material.clearcoat;
	#endif
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,n_=`#define TOON
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,i_=`#define TOON
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <gradientmap_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_toon_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_toon_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,s_=`uniform float size;
uniform float scale;
#include <common>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
#ifdef USE_POINTS_UV
	varying vec2 vUv;
	uniform mat3 uvTransform;
#endif
void main() {
	#ifdef USE_POINTS_UV
		vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	#endif
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	gl_PointSize = size;
	#ifdef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) gl_PointSize *= ( scale / - mvPosition.z );
	#endif
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <fog_vertex>
}`,r_=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <color_pars_fragment>
#include <map_particle_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_particle_fragment>
	#include <color_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,o_=`#include <common>
#include <batching_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <shadowmap_pars_vertex>
void main() {
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,a_=`uniform vec3 color;
uniform float opacity;
#include <common>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <logdepthbuf_pars_fragment>
#include <shadowmap_pars_fragment>
#include <shadowmask_pars_fragment>
void main() {
	#include <logdepthbuf_fragment>
	gl_FragColor = vec4( color, opacity * ( 1.0 - getShadowMask() ) );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,c_=`uniform float rotation;
uniform vec2 center;
#include <common>
#include <uv_pars_vertex>
#include <fog_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	vec4 mvPosition = modelViewMatrix[ 3 ];
	vec2 scale = vec2( length( modelMatrix[ 0 ].xyz ), length( modelMatrix[ 1 ].xyz ) );
	#ifndef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) scale *= - mvPosition.z;
	#endif
	vec2 alignedPosition = ( position.xy - ( center - vec2( 0.5 ) ) ) * scale;
	vec2 rotatedPosition;
	rotatedPosition.x = cos( rotation ) * alignedPosition.x - sin( rotation ) * alignedPosition.y;
	rotatedPosition.y = sin( rotation ) * alignedPosition.x + cos( rotation ) * alignedPosition.y;
	mvPosition.xy += rotatedPosition;
	gl_Position = projectionMatrix * mvPosition;
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,l_=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
}`,tt={alphahash_fragment:Pm,alphahash_pars_fragment:Im,alphamap_fragment:Lm,alphamap_pars_fragment:Dm,alphatest_fragment:Nm,alphatest_pars_fragment:Um,aomap_fragment:Om,aomap_pars_fragment:Fm,batching_pars_vertex:Bm,batching_vertex:km,begin_vertex:zm,beginnormal_vertex:Hm,bsdfs:Vm,iridescence_fragment:Gm,bumpmap_pars_fragment:Wm,clipping_planes_fragment:Xm,clipping_planes_pars_fragment:qm,clipping_planes_pars_vertex:Ym,clipping_planes_vertex:Zm,color_fragment:Km,color_pars_fragment:jm,color_pars_vertex:Jm,color_vertex:$m,common:Qm,cube_uv_reflection_fragment:eg,defaultnormal_vertex:tg,displacementmap_pars_vertex:ng,displacementmap_vertex:ig,emissivemap_fragment:sg,emissivemap_pars_fragment:rg,colorspace_fragment:og,colorspace_pars_fragment:ag,envmap_fragment:cg,envmap_common_pars_fragment:lg,envmap_pars_fragment:hg,envmap_pars_vertex:ug,envmap_physical_pars_fragment:Mg,envmap_vertex:fg,fog_vertex:dg,fog_pars_vertex:pg,fog_fragment:mg,fog_pars_fragment:gg,gradientmap_pars_fragment:_g,lightmap_pars_fragment:xg,lights_lambert_fragment:yg,lights_lambert_pars_fragment:vg,lights_pars_begin:bg,lights_toon_fragment:Sg,lights_toon_pars_fragment:Tg,lights_phong_fragment:Eg,lights_phong_pars_fragment:Ag,lights_physical_fragment:wg,lights_physical_pars_fragment:Rg,lights_fragment_begin:Cg,lights_fragment_maps:Pg,lights_fragment_end:Ig,lightprobes_pars_fragment:Lg,logdepthbuf_fragment:Dg,logdepthbuf_pars_fragment:Ng,logdepthbuf_pars_vertex:Ug,logdepthbuf_vertex:Og,map_fragment:Fg,map_pars_fragment:Bg,map_particle_fragment:kg,map_particle_pars_fragment:zg,metalnessmap_fragment:Hg,metalnessmap_pars_fragment:Vg,morphinstance_vertex:Gg,morphcolor_vertex:Wg,morphnormal_vertex:Xg,morphtarget_pars_vertex:qg,morphtarget_vertex:Yg,normal_fragment_begin:Zg,normal_fragment_maps:Kg,normal_pars_fragment:jg,normal_pars_vertex:Jg,normal_vertex:$g,normalmap_pars_fragment:Qg,clearcoat_normal_fragment_begin:e0,clearcoat_normal_fragment_maps:t0,clearcoat_pars_fragment:n0,iridescence_pars_fragment:i0,opaque_fragment:s0,packing:r0,premultiplied_alpha_fragment:o0,project_vertex:a0,dithering_fragment:c0,dithering_pars_fragment:l0,roughnessmap_fragment:h0,roughnessmap_pars_fragment:u0,shadowmap_pars_fragment:f0,shadowmap_pars_vertex:d0,shadowmap_vertex:p0,shadowmask_pars_fragment:m0,skinbase_vertex:g0,skinning_pars_vertex:_0,skinning_vertex:x0,skinnormal_vertex:y0,specularmap_fragment:v0,specularmap_pars_fragment:b0,tonemapping_fragment:M0,tonemapping_pars_fragment:S0,transmission_fragment:T0,transmission_pars_fragment:E0,uv_pars_fragment:A0,uv_pars_vertex:w0,uv_vertex:R0,worldpos_vertex:C0,background_vert:P0,background_frag:I0,backgroundCube_vert:L0,backgroundCube_frag:D0,cube_vert:N0,cube_frag:U0,depth_vert:O0,depth_frag:F0,distance_vert:B0,distance_frag:k0,equirect_vert:z0,equirect_frag:H0,linedashed_vert:V0,linedashed_frag:G0,meshbasic_vert:W0,meshbasic_frag:X0,meshlambert_vert:q0,meshlambert_frag:Y0,meshmatcap_vert:Z0,meshmatcap_frag:K0,meshnormal_vert:j0,meshnormal_frag:J0,meshphong_vert:$0,meshphong_frag:Q0,meshphysical_vert:e_,meshphysical_frag:t_,meshtoon_vert:n_,meshtoon_frag:i_,points_vert:s_,points_frag:r_,shadow_vert:o_,shadow_frag:a_,sprite_vert:c_,sprite_frag:l_},Le={common:{diffuse:{value:new Ge(16777215)},opacity:{value:1},map:{value:null},mapTransform:{value:new Ze},alphaMap:{value:null},alphaMapTransform:{value:new Ze},alphaTest:{value:0}},specularmap:{specularMap:{value:null},specularMapTransform:{value:new Ze}},envmap:{envMap:{value:null},envMapRotation:{value:new Ze},reflectivity:{value:1},ior:{value:1.5},refractionRatio:{value:.98},dfgLUT:{value:null}},aomap:{aoMap:{value:null},aoMapIntensity:{value:1},aoMapTransform:{value:new Ze}},lightmap:{lightMap:{value:null},lightMapIntensity:{value:1},lightMapTransform:{value:new Ze}},bumpmap:{bumpMap:{value:null},bumpMapTransform:{value:new Ze},bumpScale:{value:1}},normalmap:{normalMap:{value:null},normalMapTransform:{value:new Ze},normalScale:{value:new de(1,1)}},displacementmap:{displacementMap:{value:null},displacementMapTransform:{value:new Ze},displacementScale:{value:1},displacementBias:{value:0}},emissivemap:{emissiveMap:{value:null},emissiveMapTransform:{value:new Ze}},metalnessmap:{metalnessMap:{value:null},metalnessMapTransform:{value:new Ze}},roughnessmap:{roughnessMap:{value:null},roughnessMapTransform:{value:new Ze}},gradientmap:{gradientMap:{value:null}},fog:{fogDensity:{value:25e-5},fogNear:{value:1},fogFar:{value:2e3},fogColor:{value:new Ge(16777215)}},lights:{ambientLightColor:{value:[]},lightProbe:{value:[]},directionalLights:{value:[],properties:{direction:{},color:{}}},directionalLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},directionalShadowMatrix:{value:[]},spotLights:{value:[],properties:{color:{},position:{},direction:{},distance:{},coneCos:{},penumbraCos:{},decay:{}}},spotLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},spotLightMap:{value:[]},spotLightMatrix:{value:[]},pointLights:{value:[],properties:{color:{},position:{},decay:{},distance:{}}},pointLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{},shadowCameraNear:{},shadowCameraFar:{}}},pointShadowMatrix:{value:[]},hemisphereLights:{value:[],properties:{direction:{},skyColor:{},groundColor:{}}},rectAreaLights:{value:[],properties:{color:{},position:{},width:{},height:{}}},ltc_1:{value:null},ltc_2:{value:null},probesSH:{value:null},probesMin:{value:new B},probesMax:{value:new B},probesResolution:{value:new B}},points:{diffuse:{value:new Ge(16777215)},opacity:{value:1},size:{value:1},scale:{value:1},map:{value:null},alphaMap:{value:null},alphaMapTransform:{value:new Ze},alphaTest:{value:0},uvTransform:{value:new Ze}},sprite:{diffuse:{value:new Ge(16777215)},opacity:{value:1},center:{value:new de(.5,.5)},rotation:{value:0},map:{value:null},mapTransform:{value:new Ze},alphaMap:{value:null},alphaMapTransform:{value:new Ze},alphaTest:{value:0}}},si={basic:{uniforms:Qt([Le.common,Le.specularmap,Le.envmap,Le.aomap,Le.lightmap,Le.fog]),vertexShader:tt.meshbasic_vert,fragmentShader:tt.meshbasic_frag},lambert:{uniforms:Qt([Le.common,Le.specularmap,Le.envmap,Le.aomap,Le.lightmap,Le.emissivemap,Le.bumpmap,Le.normalmap,Le.displacementmap,Le.fog,Le.lights,{emissive:{value:new Ge(0)},envMapIntensity:{value:1}}]),vertexShader:tt.meshlambert_vert,fragmentShader:tt.meshlambert_frag},phong:{uniforms:Qt([Le.common,Le.specularmap,Le.envmap,Le.aomap,Le.lightmap,Le.emissivemap,Le.bumpmap,Le.normalmap,Le.displacementmap,Le.fog,Le.lights,{emissive:{value:new Ge(0)},specular:{value:new Ge(1118481)},shininess:{value:30},envMapIntensity:{value:1}}]),vertexShader:tt.meshphong_vert,fragmentShader:tt.meshphong_frag},standard:{uniforms:Qt([Le.common,Le.envmap,Le.aomap,Le.lightmap,Le.emissivemap,Le.bumpmap,Le.normalmap,Le.displacementmap,Le.roughnessmap,Le.metalnessmap,Le.fog,Le.lights,{emissive:{value:new Ge(0)},roughness:{value:1},metalness:{value:0},envMapIntensity:{value:1}}]),vertexShader:tt.meshphysical_vert,fragmentShader:tt.meshphysical_frag},toon:{uniforms:Qt([Le.common,Le.aomap,Le.lightmap,Le.emissivemap,Le.bumpmap,Le.normalmap,Le.displacementmap,Le.gradientmap,Le.fog,Le.lights,{emissive:{value:new Ge(0)}}]),vertexShader:tt.meshtoon_vert,fragmentShader:tt.meshtoon_frag},matcap:{uniforms:Qt([Le.common,Le.bumpmap,Le.normalmap,Le.displacementmap,Le.fog,{matcap:{value:null}}]),vertexShader:tt.meshmatcap_vert,fragmentShader:tt.meshmatcap_frag},points:{uniforms:Qt([Le.points,Le.fog]),vertexShader:tt.points_vert,fragmentShader:tt.points_frag},dashed:{uniforms:Qt([Le.common,Le.fog,{scale:{value:1},dashSize:{value:1},totalSize:{value:2}}]),vertexShader:tt.linedashed_vert,fragmentShader:tt.linedashed_frag},depth:{uniforms:Qt([Le.common,Le.displacementmap]),vertexShader:tt.depth_vert,fragmentShader:tt.depth_frag},normal:{uniforms:Qt([Le.common,Le.bumpmap,Le.normalmap,Le.displacementmap,{opacity:{value:1}}]),vertexShader:tt.meshnormal_vert,fragmentShader:tt.meshnormal_frag},sprite:{uniforms:Qt([Le.sprite,Le.fog]),vertexShader:tt.sprite_vert,fragmentShader:tt.sprite_frag},background:{uniforms:{uvTransform:{value:new Ze},t2D:{value:null},backgroundIntensity:{value:1}},vertexShader:tt.background_vert,fragmentShader:tt.background_frag},backgroundCube:{uniforms:{envMap:{value:null},backgroundBlurriness:{value:0},backgroundIntensity:{value:1},backgroundRotation:{value:new Ze}},vertexShader:tt.backgroundCube_vert,fragmentShader:tt.backgroundCube_frag},cube:{uniforms:{tCube:{value:null},tFlip:{value:-1},opacity:{value:1}},vertexShader:tt.cube_vert,fragmentShader:tt.cube_frag},equirect:{uniforms:{tEquirect:{value:null}},vertexShader:tt.equirect_vert,fragmentShader:tt.equirect_frag},distance:{uniforms:Qt([Le.common,Le.displacementmap,{referencePosition:{value:new B},nearDistance:{value:1},farDistance:{value:1e3}}]),vertexShader:tt.distance_vert,fragmentShader:tt.distance_frag},shadow:{uniforms:Qt([Le.lights,Le.fog,{color:{value:new Ge(0)},opacity:{value:1}}]),vertexShader:tt.shadow_vert,fragmentShader:tt.shadow_frag}};si.physical={uniforms:Qt([si.standard.uniforms,{clearcoat:{value:0},clearcoatMap:{value:null},clearcoatMapTransform:{value:new Ze},clearcoatNormalMap:{value:null},clearcoatNormalMapTransform:{value:new Ze},clearcoatNormalScale:{value:new de(1,1)},clearcoatRoughness:{value:0},clearcoatRoughnessMap:{value:null},clearcoatRoughnessMapTransform:{value:new Ze},dispersion:{value:0},iridescence:{value:0},iridescenceMap:{value:null},iridescenceMapTransform:{value:new Ze},iridescenceIOR:{value:1.3},iridescenceThicknessMinimum:{value:100},iridescenceThicknessMaximum:{value:400},iridescenceThicknessMap:{value:null},iridescenceThicknessMapTransform:{value:new Ze},sheen:{value:0},sheenColor:{value:new Ge(0)},sheenColorMap:{value:null},sheenColorMapTransform:{value:new Ze},sheenRoughness:{value:1},sheenRoughnessMap:{value:null},sheenRoughnessMapTransform:{value:new Ze},transmission:{value:0},transmissionMap:{value:null},transmissionMapTransform:{value:new Ze},transmissionSamplerSize:{value:new de},transmissionSamplerMap:{value:null},thickness:{value:0},thicknessMap:{value:null},thicknessMapTransform:{value:new Ze},attenuationDistance:{value:0},attenuationColor:{value:new Ge(0)},specularColor:{value:new Ge(1,1,1)},specularColorMap:{value:null},specularColorMapTransform:{value:new Ze},specularIntensity:{value:1},specularIntensityMap:{value:null},specularIntensityMapTransform:{value:new Ze},anisotropyVector:{value:new de},anisotropyMap:{value:null},anisotropyMapTransform:{value:new Ze}}]),vertexShader:tt.meshphysical_vert,fragmentShader:tt.meshphysical_frag};var Rc={r:0,b:0,g:0},h_=new Je,ud=new Ze;ud.set(-1,0,0,0,1,0,0,0,1);function u_(i,e,t,n,s,r){let o=new Ge(0),a=s===!0?0:1,c,l,h=null,u=0,f=null;function d(E){let S=E.isScene===!0?E.background:null;if(S&&S.isTexture){let x=E.backgroundBlurriness>0;S=e.get(S,x)}return S}function p(E){let S=!1,x=d(E);x===null?m(o,a):x&&x.isColor&&(m(x,1),S=!0);let A=i.xr.getEnvironmentBlendMode();A==="additive"?t.buffers.color.setClear(0,0,0,1,r):A==="alpha-blend"&&t.buffers.color.setClear(0,0,0,0,r),(i.autoClear||S)&&(t.buffers.depth.setTest(!0),t.buffers.depth.setMask(!0),t.buffers.color.setMask(!0),i.clear(i.autoClearColor,i.autoClearDepth,i.autoClearStencil))}function y(E,S){let x=d(S);x&&(x.isCubeTexture||x.mapping===go)?(l===void 0&&(l=new ht(new En(1,1,1),new Jt({name:"BackgroundCubeMaterial",uniforms:Es(si.backgroundCube.uniforms),vertexShader:si.backgroundCube.vertexShader,fragmentShader:si.backgroundCube.fragmentShader,side:Ft,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),l.geometry.deleteAttribute("normal"),l.geometry.deleteAttribute("uv"),l.onBeforeRender=function(A,w,I){this.matrixWorld.copyPosition(I.matrixWorld)},Object.defineProperty(l.material,"envMap",{get:function(){return this.uniforms.envMap.value}}),n.update(l)),l.material.uniforms.envMap.value=x,l.material.uniforms.backgroundBlurriness.value=S.backgroundBlurriness,l.material.uniforms.backgroundIntensity.value=S.backgroundIntensity,l.material.uniforms.backgroundRotation.value.setFromMatrix4(h_.makeRotationFromEuler(S.backgroundRotation)).transpose(),x.isCubeTexture&&x.isRenderTargetTexture===!1&&l.material.uniforms.backgroundRotation.value.premultiply(ud),l.material.toneMapped=Qe.getTransfer(x.colorSpace)!==lt,(h!==x||u!==x.version||f!==i.toneMapping)&&(l.material.needsUpdate=!0,h=x,u=x.version,f=i.toneMapping),l.layers.enableAll(),E.unshift(l,l.geometry,l.material,0,0,null)):x&&x.isTexture&&(c===void 0&&(c=new ht(new xs(2,2),new Jt({name:"BackgroundMaterial",uniforms:Es(si.background.uniforms),vertexShader:si.background.vertexShader,fragmentShader:si.background.fragmentShader,side:On,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),c.geometry.deleteAttribute("normal"),Object.defineProperty(c.material,"map",{get:function(){return this.uniforms.t2D.value}}),n.update(c)),c.material.uniforms.t2D.value=x,c.material.uniforms.backgroundIntensity.value=S.backgroundIntensity,c.material.toneMapped=Qe.getTransfer(x.colorSpace)!==lt,x.matrixAutoUpdate===!0&&x.updateMatrix(),c.material.uniforms.uvTransform.value.copy(x.matrix),(h!==x||u!==x.version||f!==i.toneMapping)&&(c.material.needsUpdate=!0,h=x,u=x.version,f=i.toneMapping),c.layers.enableAll(),E.unshift(c,c.geometry,c.material,0,0,null))}function m(E,S){E.getRGB(Rc,Ql(i)),t.buffers.color.setClear(Rc.r,Rc.g,Rc.b,S,r)}function g(){l!==void 0&&(l.geometry.dispose(),l.material.dispose(),l=void 0),c!==void 0&&(c.geometry.dispose(),c.material.dispose(),c=void 0)}return{getClearColor:function(){return o},setClearColor:function(E,S=1){o.set(E),a=S,m(o,a)},getClearAlpha:function(){return a},setClearAlpha:function(E){a=E,m(o,a)},render:p,addToRenderList:y,dispose:g}}function f_(i,e){let t=i.getParameter(i.MAX_VERTEX_ATTRIBS),n={},s=f(null),r=s,o=!1;function a(N,H,oe,ie,X){let ee=!1,Y=u(N,ie,oe,H);r!==Y&&(r=Y,l(r.object)),ee=d(N,ie,oe,X),ee&&p(N,ie,oe,X),X!==null&&e.update(X,i.ELEMENT_ARRAY_BUFFER),(ee||o)&&(o=!1,x(N,H,oe,ie),X!==null&&i.bindBuffer(i.ELEMENT_ARRAY_BUFFER,e.get(X).buffer))}function c(){return i.createVertexArray()}function l(N){return i.bindVertexArray(N)}function h(N){return i.deleteVertexArray(N)}function u(N,H,oe,ie){let X=ie.wireframe===!0,ee=n[H.id];ee===void 0&&(ee={},n[H.id]=ee);let Y=N.isInstancedMesh===!0?N.id:0,G=ee[Y];G===void 0&&(G={},ee[Y]=G);let ge=G[oe.id];ge===void 0&&(ge={},G[oe.id]=ge);let ye=ge[X];return ye===void 0&&(ye=f(c()),ge[X]=ye),ye}function f(N){let H=[],oe=[],ie=[];for(let X=0;X<t;X++)H[X]=0,oe[X]=0,ie[X]=0;return{geometry:null,program:null,wireframe:!1,newAttributes:H,enabledAttributes:oe,attributeDivisors:ie,object:N,attributes:{},index:null}}function d(N,H,oe,ie){let X=r.attributes,ee=H.attributes,Y=0,G=oe.getAttributes();for(let ge in G)if(G[ge].location>=0){let Me=X[ge],Re=ee[ge];if(Re===void 0&&(ge==="instanceMatrix"&&N.instanceMatrix&&(Re=N.instanceMatrix),ge==="instanceColor"&&N.instanceColor&&(Re=N.instanceColor)),Me===void 0||Me.attribute!==Re||Re&&Me.data!==Re.data)return!0;Y++}return r.attributesNum!==Y||r.index!==ie}function p(N,H,oe,ie){let X={},ee=H.attributes,Y=0,G=oe.getAttributes();for(let ge in G)if(G[ge].location>=0){let Me=ee[ge];Me===void 0&&(ge==="instanceMatrix"&&N.instanceMatrix&&(Me=N.instanceMatrix),ge==="instanceColor"&&N.instanceColor&&(Me=N.instanceColor));let Re={};Re.attribute=Me,Me&&Me.data&&(Re.data=Me.data),X[ge]=Re,Y++}r.attributes=X,r.attributesNum=Y,r.index=ie}function y(){let N=r.newAttributes;for(let H=0,oe=N.length;H<oe;H++)N[H]=0}function m(N){g(N,0)}function g(N,H){let oe=r.newAttributes,ie=r.enabledAttributes,X=r.attributeDivisors;oe[N]=1,ie[N]===0&&(i.enableVertexAttribArray(N),ie[N]=1),X[N]!==H&&(i.vertexAttribDivisor(N,H),X[N]=H)}function E(){let N=r.newAttributes,H=r.enabledAttributes;for(let oe=0,ie=H.length;oe<ie;oe++)H[oe]!==N[oe]&&(i.disableVertexAttribArray(oe),H[oe]=0)}function S(N,H,oe,ie,X,ee,Y){Y===!0?i.vertexAttribIPointer(N,H,oe,X,ee):i.vertexAttribPointer(N,H,oe,ie,X,ee)}function x(N,H,oe,ie){y();let X=ie.attributes,ee=oe.getAttributes(),Y=H.defaultAttributeValues;for(let G in ee){let ge=ee[G];if(ge.location>=0){let ye=X[G];if(ye===void 0&&(G==="instanceMatrix"&&N.instanceMatrix&&(ye=N.instanceMatrix),G==="instanceColor"&&N.instanceColor&&(ye=N.instanceColor)),ye!==void 0){let Me=ye.normalized,Re=ye.itemSize,ze=e.get(ye);if(ze===void 0)continue;let qe=ze.buffer,We=ze.type,pe=ze.bytesPerElement,q=We===i.INT||We===i.UNSIGNED_INT||ye.gpuType===Ga;if(ye.isInterleavedBufferAttribute){let U=ye.data,L=U.stride,P=ye.offset;if(U.isInstancedInterleavedBuffer){for(let z=0;z<ge.locationSize;z++)g(ge.location+z,U.meshPerAttribute);N.isInstancedMesh!==!0&&ie._maxInstanceCount===void 0&&(ie._maxInstanceCount=U.meshPerAttribute*U.count)}else for(let z=0;z<ge.locationSize;z++)m(ge.location+z);i.bindBuffer(i.ARRAY_BUFFER,qe);for(let z=0;z<ge.locationSize;z++)S(ge.location+z,Re/ge.locationSize,We,Me,L*pe,(P+Re/ge.locationSize*z)*pe,q)}else{if(ye.isInstancedBufferAttribute){for(let U=0;U<ge.locationSize;U++)g(ge.location+U,ye.meshPerAttribute);N.isInstancedMesh!==!0&&ie._maxInstanceCount===void 0&&(ie._maxInstanceCount=ye.meshPerAttribute*ye.count)}else for(let U=0;U<ge.locationSize;U++)m(ge.location+U);i.bindBuffer(i.ARRAY_BUFFER,qe);for(let U=0;U<ge.locationSize;U++)S(ge.location+U,Re/ge.locationSize,We,Me,Re*pe,Re/ge.locationSize*U*pe,q)}}else if(Y!==void 0){let Me=Y[G];if(Me!==void 0)switch(Me.length){case 2:i.vertexAttrib2fv(ge.location,Me);break;case 3:i.vertexAttrib3fv(ge.location,Me);break;case 4:i.vertexAttrib4fv(ge.location,Me);break;default:i.vertexAttrib1fv(ge.location,Me)}}}}E()}function A(){R();for(let N in n){let H=n[N];for(let oe in H){let ie=H[oe];for(let X in ie){let ee=ie[X];for(let Y in ee)h(ee[Y].object),delete ee[Y];delete ie[X]}}delete n[N]}}function w(N){if(n[N.id]===void 0)return;let H=n[N.id];for(let oe in H){let ie=H[oe];for(let X in ie){let ee=ie[X];for(let Y in ee)h(ee[Y].object),delete ee[Y];delete ie[X]}}delete n[N.id]}function I(N){for(let H in n){let oe=n[H];for(let ie in oe){let X=oe[ie];if(X[N.id]===void 0)continue;let ee=X[N.id];for(let Y in ee)h(ee[Y].object),delete ee[Y];delete X[N.id]}}}function v(N){for(let H in n){let oe=n[H],ie=N.isInstancedMesh===!0?N.id:0,X=oe[ie];if(X!==void 0){for(let ee in X){let Y=X[ee];for(let G in Y)h(Y[G].object),delete Y[G];delete X[ee]}delete oe[ie],Object.keys(oe).length===0&&delete n[H]}}}function R(){F(),o=!0,r!==s&&(r=s,l(r.object))}function F(){s.geometry=null,s.program=null,s.wireframe=!1}return{setup:a,reset:R,resetDefaultState:F,dispose:A,releaseStatesOfGeometry:w,releaseStatesOfObject:v,releaseStatesOfProgram:I,initAttributes:y,enableAttribute:m,disableUnusedAttributes:E}}function d_(i,e,t){let n;function s(c){n=c}function r(c,l){i.drawArrays(n,c,l),t.update(l,n,1)}function o(c,l,h){h!==0&&(i.drawArraysInstanced(n,c,l,h),t.update(l,n,h))}function a(c,l,h){if(h===0)return;e.get("WEBGL_multi_draw").multiDrawArraysWEBGL(n,c,0,l,0,h);let f=0;for(let d=0;d<h;d++)f+=l[d];t.update(f,n,1)}this.setMode=s,this.render=r,this.renderInstances=o,this.renderMultiDraw=a}function p_(i,e,t,n){let s;function r(){if(s!==void 0)return s;if(e.has("EXT_texture_filter_anisotropic")===!0){let I=e.get("EXT_texture_filter_anisotropic");s=i.getParameter(I.MAX_TEXTURE_MAX_ANISOTROPY_EXT)}else s=0;return s}function o(I){return!(I!==vn&&n.convert(I)!==i.getParameter(i.IMPLEMENTATION_COLOR_READ_FORMAT))}function a(I){let v=I===ni&&(e.has("EXT_color_buffer_half_float")||e.has("EXT_color_buffer_float"));return!(I!==hn&&n.convert(I)!==i.getParameter(i.IMPLEMENTATION_COLOR_READ_TYPE)&&I!==nn&&!v)}function c(I){if(I==="highp"){if(i.getShaderPrecisionFormat(i.VERTEX_SHADER,i.HIGH_FLOAT).precision>0&&i.getShaderPrecisionFormat(i.FRAGMENT_SHADER,i.HIGH_FLOAT).precision>0)return"highp";I="mediump"}return I==="mediump"&&i.getShaderPrecisionFormat(i.VERTEX_SHADER,i.MEDIUM_FLOAT).precision>0&&i.getShaderPrecisionFormat(i.FRAGMENT_SHADER,i.MEDIUM_FLOAT).precision>0?"mediump":"lowp"}let l=t.precision!==void 0?t.precision:"highp",h=c(l);h!==l&&(ke("WebGLRenderer:",l,"not supported, using",h,"instead."),l=h);let u=t.logarithmicDepthBuffer===!0,f=t.reversedDepthBuffer===!0&&e.has("EXT_clip_control");t.reversedDepthBuffer===!0&&f===!1&&ke("WebGLRenderer: Unable to use reversed depth buffer due to missing EXT_clip_control extension. Fallback to default depth buffer.");let d=i.getParameter(i.MAX_TEXTURE_IMAGE_UNITS),p=i.getParameter(i.MAX_VERTEX_TEXTURE_IMAGE_UNITS),y=i.getParameter(i.MAX_TEXTURE_SIZE),m=i.getParameter(i.MAX_CUBE_MAP_TEXTURE_SIZE),g=i.getParameter(i.MAX_VERTEX_ATTRIBS),E=i.getParameter(i.MAX_VERTEX_UNIFORM_VECTORS),S=i.getParameter(i.MAX_VARYING_VECTORS),x=i.getParameter(i.MAX_FRAGMENT_UNIFORM_VECTORS),A=i.getParameter(i.MAX_SAMPLES),w=i.getParameter(i.SAMPLES);return{isWebGL2:!0,getMaxAnisotropy:r,getMaxPrecision:c,textureFormatReadable:o,textureTypeReadable:a,precision:l,logarithmicDepthBuffer:u,reversedDepthBuffer:f,maxTextures:d,maxVertexTextures:p,maxTextureSize:y,maxCubemapSize:m,maxAttributes:g,maxVertexUniforms:E,maxVaryings:S,maxFragmentUniforms:x,maxSamples:A,samples:w}}function m_(i){let e=this,t=null,n=0,s=!1,r=!1,o=new Sn,a=new Ze,c={value:null,needsUpdate:!1};this.uniform=c,this.numPlanes=0,this.numIntersection=0,this.init=function(u,f){let d=u.length!==0||f||n!==0||s;return s=f,n=u.length,d},this.beginShadows=function(){r=!0,h(null)},this.endShadows=function(){r=!1},this.setGlobalState=function(u,f){t=h(u,f,0)},this.setState=function(u,f,d){let p=u.clippingPlanes,y=u.clipIntersection,m=u.clipShadows,g=i.get(u);if(!s||p===null||p.length===0||r&&!m)r?h(null):l();else{let E=r?0:n,S=E*4,x=g.clippingState||null;c.value=x,x=h(p,f,S,d);for(let A=0;A!==S;++A)x[A]=t[A];g.clippingState=x,this.numIntersection=y?this.numPlanes:0,this.numPlanes+=E}};function l(){c.value!==t&&(c.value=t,c.needsUpdate=n>0),e.numPlanes=n,e.numIntersection=0}function h(u,f,d,p){let y=u!==null?u.length:0,m=null;if(y!==0){if(m=c.value,p!==!0||m===null){let g=d+y*4,E=f.matrixWorldInverse;a.getNormalMatrix(E),(m===null||m.length<g)&&(m=new Float32Array(g));for(let S=0,x=d;S!==y;++S,x+=4)o.copy(u[S]).applyMatrix4(E,a),o.normal.toArray(m,x),m[x+3]=o.constant}c.value=m,c.needsUpdate=!0}return e.numPlanes=y,e.numIntersection=0,m}}var $i=4,Vf=[.125,.215,.35,.446,.526,.582],As=20,g_=256,To=new ti,Gf=new Ge,ch=null,lh=0,hh=0,uh=!1,__=new B,Qi=class{constructor(e){this._renderer=e,this._pingPongRenderTarget=null,this._lodMax=0,this._cubeSize=0,this._sizeLods=[],this._sigmas=[],this._lodMeshes=[],this._backgroundBox=null,this._cubemapMaterial=null,this._equirectMaterial=null,this._blurMaterial=null,this._ggxMaterial=null}fromScene(e,t=0,n=.1,s=100,r={}){let{size:o=256,position:a=__}=r;ch=this._renderer.getRenderTarget(),lh=this._renderer.getActiveCubeFace(),hh=this._renderer.getActiveMipmapLevel(),uh=this._renderer.xr.enabled,this._renderer.xr.enabled=!1,this._setSize(o);let c=this._allocateTargets();return c.depthBuffer=!0,this._sceneToCubeUV(e,n,s,c,a),t>0&&this._blur(c,0,0,t),this._applyPMREM(c),this._cleanup(c),c}fromEquirectangular(e,t=null){return this._fromTexture(e,t)}fromCubemap(e,t=null){return this._fromTexture(e,t)}compileCubemapShader(){this._cubemapMaterial===null&&(this._cubemapMaterial=qf(),this._compileMaterial(this._cubemapMaterial))}compileEquirectangularShader(){this._equirectMaterial===null&&(this._equirectMaterial=Xf(),this._compileMaterial(this._equirectMaterial))}dispose(){this._dispose(),this._cubemapMaterial!==null&&this._cubemapMaterial.dispose(),this._equirectMaterial!==null&&this._equirectMaterial.dispose(),this._backgroundBox!==null&&(this._backgroundBox.geometry.dispose(),this._backgroundBox.material.dispose())}_setSize(e){this._lodMax=Math.floor(Math.log2(e)),this._cubeSize=Math.pow(2,this._lodMax)}_dispose(){this._blurMaterial!==null&&this._blurMaterial.dispose(),this._ggxMaterial!==null&&this._ggxMaterial.dispose(),this._pingPongRenderTarget!==null&&this._pingPongRenderTarget.dispose();for(let e=0;e<this._lodMeshes.length;e++)this._lodMeshes[e].geometry.dispose()}_cleanup(e){this._renderer.setRenderTarget(ch,lh,hh),this._renderer.xr.enabled=uh,e.scissorTest=!1,mr(e,0,0,e.width,e.height)}_fromTexture(e,t){e.mapping===Ki||e.mapping===Ms?this._setSize(e.image.length===0?16:e.image[0].width||e.image[0].image.width):this._setSize(e.image.width/4),ch=this._renderer.getRenderTarget(),lh=this._renderer.getActiveCubeFace(),hh=this._renderer.getActiveMipmapLevel(),uh=this._renderer.xr.enabled,this._renderer.xr.enabled=!1;let n=t||this._allocateTargets();return this._textureToCubeUV(e,n),this._applyPMREM(n),this._cleanup(n),n}_allocateTargets(){let e=3*Math.max(this._cubeSize,112),t=4*this._cubeSize,n={magFilter:bt,minFilter:bt,generateMipmaps:!1,type:ni,format:vn,colorSpace:Wt,depthBuffer:!1},s=Wf(e,t,n);if(this._pingPongRenderTarget===null||this._pingPongRenderTarget.width!==e||this._pingPongRenderTarget.height!==t){this._pingPongRenderTarget!==null&&this._dispose(),this._pingPongRenderTarget=Wf(e,t,n);let{_lodMax:r}=this;({lodMeshes:this._lodMeshes,sizeLods:this._sizeLods,sigmas:this._sigmas}=x_(r)),this._blurMaterial=v_(r,e,t),this._ggxMaterial=y_(r,e,t)}return s}_compileMaterial(e){let t=new ht(new wt,e);this._renderer.compile(t,To)}_sceneToCubeUV(e,t,n,s,r){let c=new Ct(90,1,t,n),l=[1,-1,1,1,1,1],h=[1,1,1,-1,-1,-1],u=this._renderer,f=u.autoClear,d=u.toneMapping;u.getClearColor(Gf),u.toneMapping=Gn,u.autoClear=!1,u.state.buffers.depth.getReversed()&&(u.setRenderTarget(s),u.clearDepth(),u.setRenderTarget(null)),this._backgroundBox===null&&(this._backgroundBox=new ht(new En,new Ot({name:"PMREM.Background",side:Ft,depthWrite:!1,depthTest:!1})));let y=this._backgroundBox,m=y.material,g=!1,E=e.background;E?E.isColor&&(m.color.copy(E),e.background=null,g=!0):(m.color.copy(Gf),g=!0);for(let S=0;S<6;S++){let x=S%3;x===0?(c.up.set(0,l[S],0),c.position.set(r.x,r.y,r.z),c.lookAt(r.x+h[S],r.y,r.z)):x===1?(c.up.set(0,0,l[S]),c.position.set(r.x,r.y,r.z),c.lookAt(r.x,r.y+h[S],r.z)):(c.up.set(0,l[S],0),c.position.set(r.x,r.y,r.z),c.lookAt(r.x,r.y,r.z+h[S]));let A=this._cubeSize;mr(s,x*A,S>2?A:0,A,A),u.setRenderTarget(s),g&&u.render(y,c),u.render(e,c)}u.toneMapping=d,u.autoClear=f,e.background=E}_textureToCubeUV(e,t){let n=this._renderer,s=e.mapping===Ki||e.mapping===Ms;s?(this._cubemapMaterial===null&&(this._cubemapMaterial=qf()),this._cubemapMaterial.uniforms.flipEnvMap.value=e.isRenderTargetTexture===!1?-1:1):this._equirectMaterial===null&&(this._equirectMaterial=Xf());let r=s?this._cubemapMaterial:this._equirectMaterial,o=this._lodMeshes[0];o.material=r;let a=r.uniforms;a.envMap.value=e;let c=this._cubeSize;mr(t,0,0,3*c,2*c),n.setRenderTarget(t),n.render(o,To)}_applyPMREM(e){let t=this._renderer,n=t.autoClear;t.autoClear=!1;let s=this._lodMeshes.length;for(let r=1;r<s;r++)this._applyGGXFilter(e,r-1,r);t.autoClear=n}_applyGGXFilter(e,t,n){let s=this._renderer,r=this._pingPongRenderTarget,o=this._ggxMaterial,a=this._lodMeshes[n];a.material=o;let c=o.uniforms,l=n/(this._lodMeshes.length-1),h=t/(this._lodMeshes.length-1),u=Math.sqrt(l*l-h*h),f=0+l*1.25,d=u*f,{_lodMax:p}=this,y=this._sizeLods[n],m=3*y*(n>p-$i?n-p+$i:0),g=4*(this._cubeSize-y);c.envMap.value=e.texture,c.roughness.value=d,c.mipInt.value=p-t,mr(r,m,g,3*y,2*y),s.setRenderTarget(r),s.render(a,To),c.envMap.value=r.texture,c.roughness.value=0,c.mipInt.value=p-n,mr(e,m,g,3*y,2*y),s.setRenderTarget(e),s.render(a,To)}_blur(e,t,n,s,r){let o=this._pingPongRenderTarget;this._halfBlur(e,o,t,n,s,"latitudinal",r),this._halfBlur(o,e,n,n,s,"longitudinal",r)}_halfBlur(e,t,n,s,r,o,a){let c=this._renderer,l=this._blurMaterial;o!=="latitudinal"&&o!=="longitudinal"&&Ke("blur direction must be either latitudinal or longitudinal!");let h=3,u=this._lodMeshes[s];u.material=l;let f=l.uniforms,d=this._sizeLods[n]-1,p=isFinite(r)?Math.PI/(2*d):2*Math.PI/(2*As-1),y=r/p,m=isFinite(r)?1+Math.floor(h*y):As;m>As&&ke(`sigmaRadians, ${r}, is too large and will clip, as it requested ${m} samples when the maximum is set to ${As}`);let g=[],E=0;for(let I=0;I<As;++I){let v=I/y,R=Math.exp(-v*v/2);g.push(R),I===0?E+=R:I<m&&(E+=2*R)}for(let I=0;I<g.length;I++)g[I]=g[I]/E;f.envMap.value=e.texture,f.samples.value=m,f.weights.value=g,f.latitudinal.value=o==="latitudinal",a&&(f.poleAxis.value=a);let{_lodMax:S}=this;f.dTheta.value=p,f.mipInt.value=S-n;let x=this._sizeLods[s],A=3*x*(s>S-$i?s-S+$i:0),w=4*(this._cubeSize-x);mr(t,A,w,3*x,2*x),c.setRenderTarget(t),c.render(u,To)}};function x_(i){let e=[],t=[],n=[],s=i,r=i-$i+1+Vf.length;for(let o=0;o<r;o++){let a=Math.pow(2,s);e.push(a);let c=1/a;o>i-$i?c=Vf[o-i+$i-1]:o===0&&(c=0),t.push(c);let l=1/(a-2),h=-l,u=1+l,f=[h,h,u,h,u,u,h,h,u,u,h,u],d=6,p=6,y=3,m=2,g=1,E=new Float32Array(y*p*d),S=new Float32Array(m*p*d),x=new Float32Array(g*p*d);for(let w=0;w<d;w++){let I=w%3*2/3-1,v=w>2?0:-1,R=[I,v,0,I+2/3,v,0,I+2/3,v+1,0,I,v,0,I+2/3,v+1,0,I,v+1,0];E.set(R,y*p*w),S.set(f,m*p*w);let F=[w,w,w,w,w,w];x.set(F,g*p*w)}let A=new wt;A.setAttribute("position",new yt(E,y)),A.setAttribute("uv",new yt(S,m)),A.setAttribute("faceIndex",new yt(x,g)),n.push(new ht(A,null)),s>$i&&s--}return{lodMeshes:n,sizeLods:e,sigmas:t}}function Wf(i,e,t){let n=new jt(i,e,t);return n.texture.mapping=go,n.texture.name="PMREM.cubeUv",n.scissorTest=!0,n}function mr(i,e,t,n,s){i.viewport.set(e,t,n,s),i.scissor.set(e,t,n,s)}function y_(i,e,t){return new Jt({name:"PMREMGGXConvolution",defines:{GGX_SAMPLES:g_,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/t,CUBEUV_MAX_MIP:`${i}.0`},uniforms:{envMap:{value:null},roughness:{value:0},mipInt:{value:0}},vertexShader:Ic(),fragmentShader:`

			precision highp float;
			precision highp int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform float roughness;
			uniform float mipInt;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			#define PI 3.14159265359

			// Van der Corput radical inverse
			float radicalInverse_VdC(uint bits) {
				bits = (bits << 16u) | (bits >> 16u);
				bits = ((bits & 0x55555555u) << 1u) | ((bits & 0xAAAAAAAAu) >> 1u);
				bits = ((bits & 0x33333333u) << 2u) | ((bits & 0xCCCCCCCCu) >> 2u);
				bits = ((bits & 0x0F0F0F0Fu) << 4u) | ((bits & 0xF0F0F0F0u) >> 4u);
				bits = ((bits & 0x00FF00FFu) << 8u) | ((bits & 0xFF00FF00u) >> 8u);
				return float(bits) * 2.3283064365386963e-10; // / 0x100000000
			}

			// Hammersley sequence
			vec2 hammersley(uint i, uint N) {
				return vec2(float(i) / float(N), radicalInverse_VdC(i));
			}

			// GGX VNDF importance sampling (Eric Heitz 2018)
			// "Sampling the GGX Distribution of Visible Normals"
			// https://jcgt.org/published/0007/04/01/
			vec3 importanceSampleGGX_VNDF(vec2 Xi, vec3 V, float roughness) {
				float alpha = roughness * roughness;

				// Section 4.1: Orthonormal basis
				vec3 T1 = vec3(1.0, 0.0, 0.0);
				vec3 T2 = cross(V, T1);

				// Section 4.2: Parameterization of projected area
				float r = sqrt(Xi.x);
				float phi = 2.0 * PI * Xi.y;
				float t1 = r * cos(phi);
				float t2 = r * sin(phi);
				float s = 0.5 * (1.0 + V.z);
				t2 = (1.0 - s) * sqrt(1.0 - t1 * t1) + s * t2;

				// Section 4.3: Reprojection onto hemisphere
				vec3 Nh = t1 * T1 + t2 * T2 + sqrt(max(0.0, 1.0 - t1 * t1 - t2 * t2)) * V;

				// Section 3.4: Transform back to ellipsoid configuration
				return normalize(vec3(alpha * Nh.x, alpha * Nh.y, max(0.0, Nh.z)));
			}

			void main() {
				vec3 N = normalize(vOutputDirection);
				vec3 V = N; // Assume view direction equals normal for pre-filtering

				vec3 prefilteredColor = vec3(0.0);
				float totalWeight = 0.0;

				// For very low roughness, just sample the environment directly
				if (roughness < 0.001) {
					gl_FragColor = vec4(bilinearCubeUV(envMap, N, mipInt), 1.0);
					return;
				}

				// Tangent space basis for VNDF sampling
				vec3 up = abs(N.z) < 0.999 ? vec3(0.0, 0.0, 1.0) : vec3(1.0, 0.0, 0.0);
				vec3 tangent = normalize(cross(up, N));
				vec3 bitangent = cross(N, tangent);

				for(uint i = 0u; i < uint(GGX_SAMPLES); i++) {
					vec2 Xi = hammersley(i, uint(GGX_SAMPLES));

					// For PMREM, V = N, so in tangent space V is always (0, 0, 1)
					vec3 H_tangent = importanceSampleGGX_VNDF(Xi, vec3(0.0, 0.0, 1.0), roughness);

					// Transform H back to world space
					vec3 H = normalize(tangent * H_tangent.x + bitangent * H_tangent.y + N * H_tangent.z);
					vec3 L = normalize(2.0 * dot(V, H) * H - V);

					float NdotL = max(dot(N, L), 0.0);

					if(NdotL > 0.0) {
						// Sample environment at fixed mip level
						// VNDF importance sampling handles the distribution filtering
						vec3 sampleColor = bilinearCubeUV(envMap, L, mipInt);

						// Weight by NdotL for the split-sum approximation
						// VNDF PDF naturally accounts for the visible microfacet distribution
						prefilteredColor += sampleColor * NdotL;
						totalWeight += NdotL;
					}
				}

				if (totalWeight > 0.0) {
					prefilteredColor = prefilteredColor / totalWeight;
				}

				gl_FragColor = vec4(prefilteredColor, 1.0);
			}
		`,blending:xn,depthTest:!1,depthWrite:!1})}function v_(i,e,t){let n=new Float32Array(As),s=new B(0,1,0);return new Jt({name:"SphericalGaussianBlur",defines:{n:As,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/t,CUBEUV_MAX_MIP:`${i}.0`},uniforms:{envMap:{value:null},samples:{value:1},weights:{value:n},latitudinal:{value:!1},dTheta:{value:0},mipInt:{value:0},poleAxis:{value:s}},vertexShader:Ic(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform int samples;
			uniform float weights[ n ];
			uniform bool latitudinal;
			uniform float dTheta;
			uniform float mipInt;
			uniform vec3 poleAxis;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			vec3 getSample( float theta, vec3 axis ) {

				float cosTheta = cos( theta );
				// Rodrigues' axis-angle rotation
				vec3 sampleDirection = vOutputDirection * cosTheta
					+ cross( axis, vOutputDirection ) * sin( theta )
					+ axis * dot( axis, vOutputDirection ) * ( 1.0 - cosTheta );

				return bilinearCubeUV( envMap, sampleDirection, mipInt );

			}

			void main() {

				vec3 axis = latitudinal ? poleAxis : cross( poleAxis, vOutputDirection );

				if ( all( equal( axis, vec3( 0.0 ) ) ) ) {

					axis = vec3( vOutputDirection.z, 0.0, - vOutputDirection.x );

				}

				axis = normalize( axis );

				gl_FragColor = vec4( 0.0, 0.0, 0.0, 1.0 );
				gl_FragColor.rgb += weights[ 0 ] * getSample( 0.0, axis );

				for ( int i = 1; i < n; i++ ) {

					if ( i >= samples ) {

						break;

					}

					float theta = dTheta * float( i );
					gl_FragColor.rgb += weights[ i ] * getSample( -1.0 * theta, axis );
					gl_FragColor.rgb += weights[ i ] * getSample( theta, axis );

				}

			}
		`,blending:xn,depthTest:!1,depthWrite:!1})}function Xf(){return new Jt({name:"EquirectangularToCubeUV",uniforms:{envMap:{value:null}},vertexShader:Ic(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;

			#include <common>

			void main() {

				vec3 outputDirection = normalize( vOutputDirection );
				vec2 uv = equirectUv( outputDirection );

				gl_FragColor = vec4( texture2D ( envMap, uv ).rgb, 1.0 );

			}
		`,blending:xn,depthTest:!1,depthWrite:!1})}function qf(){return new Jt({name:"CubemapToCubeUV",uniforms:{envMap:{value:null},flipEnvMap:{value:-1}},vertexShader:Ic(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			uniform float flipEnvMap;

			varying vec3 vOutputDirection;

			uniform samplerCube envMap;

			void main() {

				gl_FragColor = textureCube( envMap, vec3( flipEnvMap * vOutputDirection.x, vOutputDirection.yz ) );

			}
		`,blending:xn,depthTest:!1,depthWrite:!1})}function Ic(){return`

		precision mediump float;
		precision mediump int;

		attribute float faceIndex;

		varying vec3 vOutputDirection;

		// RH coordinate system; PMREM face-indexing convention
		vec3 getDirection( vec2 uv, float face ) {

			uv = 2.0 * uv - 1.0;

			vec3 direction = vec3( uv, 1.0 );

			if ( face == 0.0 ) {

				direction = direction.zyx; // ( 1, v, u ) pos x

			} else if ( face == 1.0 ) {

				direction = direction.xzy;
				direction.xz *= -1.0; // ( -u, 1, -v ) pos y

			} else if ( face == 2.0 ) {

				direction.x *= -1.0; // ( -u, v, 1 ) pos z

			} else if ( face == 3.0 ) {

				direction = direction.zyx;
				direction.xz *= -1.0; // ( -1, v, -u ) neg x

			} else if ( face == 4.0 ) {

				direction = direction.xzy;
				direction.xy *= -1.0; // ( -u, -1, v ) neg y

			} else if ( face == 5.0 ) {

				direction.z *= -1.0; // ( u, v, -1 ) neg z

			}

			return direction;

		}

		void main() {

			vOutputDirection = getDirection( uv, faceIndex );
			gl_Position = vec4( position, 1.0 );

		}
	`}var Pc=class extends jt{constructor(e=1,t={}){super(e,e,t),this.isWebGLCubeRenderTarget=!0;let n={width:e,height:e,depth:1},s=[n,n,n,n,n,n];this.texture=new $r(s),this._setTextureOptions(t),this.texture.isRenderTargetTexture=!0}fromEquirectangularTexture(e,t){this.texture.type=t.type,this.texture.colorSpace=t.colorSpace,this.texture.generateMipmaps=t.generateMipmaps,this.texture.minFilter=t.minFilter,this.texture.magFilter=t.magFilter;let n={uniforms:{tEquirect:{value:null}},vertexShader:`

				varying vec3 vWorldDirection;

				vec3 transformDirection( in vec3 dir, in mat4 matrix ) {

					return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );

				}

				void main() {

					vWorldDirection = transformDirection( position, modelMatrix );

					#include <begin_vertex>
					#include <project_vertex>

				}
			`,fragmentShader:`

				uniform sampler2D tEquirect;

				varying vec3 vWorldDirection;

				#include <common>

				void main() {

					vec3 direction = normalize( vWorldDirection );

					vec2 sampleUV = equirectUv( direction );

					gl_FragColor = texture2D( tEquirect, sampleUV );

				}
			`},s=new En(5,5,5),r=new Jt({name:"CubemapFromEquirect",uniforms:Es(n.uniforms),vertexShader:n.vertexShader,fragmentShader:n.fragmentShader,side:Ft,blending:xn});r.uniforms.tEquirect.value=t;let o=new ht(s,r),a=t.minFilter;return t.minFilter===yn&&(t.minFilter=bt),new Ba(1,10,this).update(e,o),t.minFilter=a,o.geometry.dispose(),o.material.dispose(),this}clear(e,t=!0,n=!0,s=!0){let r=e.getRenderTarget();for(let o=0;o<6;o++)e.setRenderTarget(this,o),e.clear(t,n,s);e.setRenderTarget(r)}};function b_(i){let e=new WeakMap,t=new WeakMap,n=null;function s(f,d=!1){return f==null?null:d?o(f):r(f)}function r(f){if(f&&f.isTexture){let d=f.mapping;if(d===cr||d===Ha)if(e.has(f)){let p=e.get(f).texture;return a(p,f.mapping)}else{let p=f.image;if(p&&p.height>0){let y=new Pc(p.height);return y.fromEquirectangularTexture(i,f),e.set(f,y),f.addEventListener("dispose",l),a(y.texture,f.mapping)}else return null}}return f}function o(f){if(f&&f.isTexture){let d=f.mapping,p=d===cr||d===Ha,y=d===Ki||d===Ms;if(p||y){let m=t.get(f),g=m!==void 0?m.texture.pmremVersion:0;if(f.isRenderTargetTexture&&f.pmremVersion!==g)return n===null&&(n=new Qi(i)),m=p?n.fromEquirectangular(f,m):n.fromCubemap(f,m),m.texture.pmremVersion=f.pmremVersion,t.set(f,m),m.texture;if(m!==void 0)return m.texture;{let E=f.image;return p&&E&&E.height>0||y&&E&&c(E)?(n===null&&(n=new Qi(i)),m=p?n.fromEquirectangular(f):n.fromCubemap(f),m.texture.pmremVersion=f.pmremVersion,t.set(f,m),f.addEventListener("dispose",h),m.texture):null}}}return f}function a(f,d){return d===cr?f.mapping=Ki:d===Ha&&(f.mapping=Ms),f}function c(f){let d=0,p=6;for(let y=0;y<p;y++)f[y]!==void 0&&d++;return d===p}function l(f){let d=f.target;d.removeEventListener("dispose",l);let p=e.get(d);p!==void 0&&(e.delete(d),p.dispose())}function h(f){let d=f.target;d.removeEventListener("dispose",h);let p=t.get(d);p!==void 0&&(t.delete(d),p.dispose())}function u(){e=new WeakMap,t=new WeakMap,n!==null&&(n.dispose(),n=null)}return{get:s,dispose:u}}function M_(i){let e={};function t(n){if(e[n]!==void 0)return e[n];let s=i.getExtension(n);return e[n]=s,s}return{has:function(n){return t(n)!==null},init:function(){t("EXT_color_buffer_float"),t("WEBGL_clip_cull_distance"),t("OES_texture_float_linear"),t("EXT_color_buffer_half_float"),t("WEBGL_multisampled_render_to_texture"),t("WEBGL_render_shared_exponent")},get:function(n){let s=t(n);return s===null&&ls("WebGLRenderer: "+n+" extension not supported."),s}}}function S_(i,e,t,n){let s={},r=new WeakMap;function o(u){let f=u.target;f.index!==null&&e.remove(f.index);for(let p in f.attributes)e.remove(f.attributes[p]);f.removeEventListener("dispose",o),delete s[f.id];let d=r.get(f);d&&(e.remove(d),r.delete(f)),n.releaseStatesOfGeometry(f),f.isInstancedBufferGeometry===!0&&delete f._maxInstanceCount,t.memory.geometries--}function a(u,f){return s[f.id]===!0||(f.addEventListener("dispose",o),s[f.id]=!0,t.memory.geometries++),f}function c(u){let f=u.attributes;for(let d in f)e.update(f[d],i.ARRAY_BUFFER)}function l(u){let f=[],d=u.index,p=u.attributes.position,y=0;if(p===void 0)return;if(d!==null){let E=d.array;y=d.version;for(let S=0,x=E.length;S<x;S+=3){let A=E[S+0],w=E[S+1],I=E[S+2];f.push(A,w,w,I,I,A)}}else{let E=p.array;y=p.version;for(let S=0,x=E.length/3-1;S<x;S+=3){let A=S+0,w=S+1,I=S+2;f.push(A,w,w,I,I,A)}}let m=new(p.count>=65535?Xr:Wr)(f,1);m.version=y;let g=r.get(u);g&&e.remove(g),r.set(u,m)}function h(u){let f=r.get(u);if(f){let d=u.index;d!==null&&f.version<d.version&&l(u)}else l(u);return r.get(u)}return{get:a,update:c,getWireframeAttribute:h}}function T_(i,e,t){let n;function s(u){n=u}let r,o;function a(u){r=u.type,o=u.bytesPerElement}function c(u,f){i.drawElements(n,f,r,u*o),t.update(f,n,1)}function l(u,f,d){d!==0&&(i.drawElementsInstanced(n,f,r,u*o,d),t.update(f,n,d))}function h(u,f,d){if(d===0)return;e.get("WEBGL_multi_draw").multiDrawElementsWEBGL(n,f,0,r,u,0,d);let y=0;for(let m=0;m<d;m++)y+=f[m];t.update(y,n,1)}this.setMode=s,this.setIndex=a,this.render=c,this.renderInstances=l,this.renderMultiDraw=h}function E_(i){let e={geometries:0,textures:0},t={frame:0,calls:0,triangles:0,points:0,lines:0};function n(r,o,a){switch(t.calls++,o){case i.TRIANGLES:t.triangles+=a*(r/3);break;case i.LINES:t.lines+=a*(r/2);break;case i.LINE_STRIP:t.lines+=a*(r-1);break;case i.LINE_LOOP:t.lines+=a*r;break;case i.POINTS:t.points+=a*r;break;default:Ke("WebGLInfo: Unknown draw mode:",o);break}}function s(){t.calls=0,t.triangles=0,t.points=0,t.lines=0}return{memory:e,render:t,programs:null,autoReset:!0,reset:s,update:n}}function A_(i,e,t){let n=new WeakMap,s=new ft;function r(o,a,c){let l=o.morphTargetInfluences,h=a.morphAttributes.position||a.morphAttributes.normal||a.morphAttributes.color,u=h!==void 0?h.length:0,f=n.get(a);if(f===void 0||f.count!==u){let R=function(){I.dispose(),n.delete(a),a.removeEventListener("dispose",R)};f!==void 0&&f.texture.dispose();let d=a.morphAttributes.position!==void 0,p=a.morphAttributes.normal!==void 0,y=a.morphAttributes.color!==void 0,m=a.morphAttributes.position||[],g=a.morphAttributes.normal||[],E=a.morphAttributes.color||[],S=0;d===!0&&(S=1),p===!0&&(S=2),y===!0&&(S=3);let x=a.attributes.position.count*S,A=1;x>e.maxTextureSize&&(A=Math.ceil(x/e.maxTextureSize),x=e.maxTextureSize);let w=new Float32Array(x*A*4*u),I=new Vr(w,x,A,u);I.type=nn,I.needsUpdate=!0;let v=S*4;for(let F=0;F<u;F++){let N=m[F],H=g[F],oe=E[F],ie=x*A*4*F;for(let X=0;X<N.count;X++){let ee=X*v;d===!0&&(s.fromBufferAttribute(N,X),w[ie+ee+0]=s.x,w[ie+ee+1]=s.y,w[ie+ee+2]=s.z,w[ie+ee+3]=0),p===!0&&(s.fromBufferAttribute(H,X),w[ie+ee+4]=s.x,w[ie+ee+5]=s.y,w[ie+ee+6]=s.z,w[ie+ee+7]=0),y===!0&&(s.fromBufferAttribute(oe,X),w[ie+ee+8]=s.x,w[ie+ee+9]=s.y,w[ie+ee+10]=s.z,w[ie+ee+11]=oe.itemSize===4?s.w:1)}}f={count:u,texture:I,size:new de(x,A)},n.set(a,f),a.addEventListener("dispose",R)}if(o.isInstancedMesh===!0&&o.morphTexture!==null)c.getUniforms().setValue(i,"morphTexture",o.morphTexture,t);else{let d=0;for(let y=0;y<l.length;y++)d+=l[y];let p=a.morphTargetsRelative?1:1-d;c.getUniforms().setValue(i,"morphTargetBaseInfluence",p),c.getUniforms().setValue(i,"morphTargetInfluences",l)}c.getUniforms().setValue(i,"morphTargetsTexture",f.texture,t),c.getUniforms().setValue(i,"morphTargetsTextureSize",f.size)}return{update:r}}function w_(i,e,t,n,s){let r=new WeakMap;function o(l){let h=s.render.frame,u=l.geometry,f=e.get(l,u);if(r.get(f)!==h&&(e.update(f),r.set(f,h)),l.isInstancedMesh&&(l.hasEventListener("dispose",c)===!1&&l.addEventListener("dispose",c),r.get(l)!==h&&(t.update(l.instanceMatrix,i.ARRAY_BUFFER),l.instanceColor!==null&&t.update(l.instanceColor,i.ARRAY_BUFFER),r.set(l,h))),l.isSkinnedMesh){let d=l.skeleton;r.get(d)!==h&&(d.update(),r.set(d,h))}return f}function a(){r=new WeakMap}function c(l){let h=l.target;h.removeEventListener("dispose",c),n.releaseStatesOfObject(h),t.remove(h.instanceMatrix),h.instanceColor!==null&&t.remove(h.instanceColor)}return{update:o,dispose:a}}var R_={[Fl]:"LINEAR_TONE_MAPPING",[Bl]:"REINHARD_TONE_MAPPING",[kl]:"CINEON_TONE_MAPPING",[bs]:"ACES_FILMIC_TONE_MAPPING",[Hl]:"AGX_TONE_MAPPING",[Vl]:"NEUTRAL_TONE_MAPPING",[zl]:"CUSTOM_TONE_MAPPING"};function C_(i,e,t,n,s,r){let o=new jt(e,t,{type:i,depthBuffer:s,stencilBuffer:r,samples:n?4:0,depthTexture:s?new xi(e,t):void 0}),a=new jt(e,t,{type:ni,depthBuffer:!1,stencilBuffer:!1}),c=new wt;c.setAttribute("position",new Mt([-1,3,0,-1,-1,0,3,-1,0],3)),c.setAttribute("uv",new Mt([0,2,0,0,2,0],2));let l=new Ra({uniforms:{tDiffuse:{value:null}},vertexShader:`
			precision highp float;

			uniform mat4 modelViewMatrix;
			uniform mat4 projectionMatrix;

			attribute vec3 position;
			attribute vec2 uv;

			varying vec2 vUv;

			void main() {
				vUv = uv;
				gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
			}`,fragmentShader:`
			precision highp float;

			uniform sampler2D tDiffuse;

			varying vec2 vUv;

			#include <tonemapping_pars_fragment>
			#include <colorspace_pars_fragment>

			void main() {
				gl_FragColor = texture2D( tDiffuse, vUv );

				#ifdef LINEAR_TONE_MAPPING
					gl_FragColor.rgb = LinearToneMapping( gl_FragColor.rgb );
				#elif defined( REINHARD_TONE_MAPPING )
					gl_FragColor.rgb = ReinhardToneMapping( gl_FragColor.rgb );
				#elif defined( CINEON_TONE_MAPPING )
					gl_FragColor.rgb = CineonToneMapping( gl_FragColor.rgb );
				#elif defined( ACES_FILMIC_TONE_MAPPING )
					gl_FragColor.rgb = ACESFilmicToneMapping( gl_FragColor.rgb );
				#elif defined( AGX_TONE_MAPPING )
					gl_FragColor.rgb = AgXToneMapping( gl_FragColor.rgb );
				#elif defined( NEUTRAL_TONE_MAPPING )
					gl_FragColor.rgb = NeutralToneMapping( gl_FragColor.rgb );
				#elif defined( CUSTOM_TONE_MAPPING )
					gl_FragColor.rgb = CustomToneMapping( gl_FragColor.rgb );
				#endif

				#ifdef SRGB_TRANSFER
					gl_FragColor = sRGBTransferOETF( gl_FragColor );
				#endif
			}`,depthTest:!1,depthWrite:!1}),h=new ht(c,l),u=new ti(-1,1,1,-1,0,1),f=null,d=null,p=!1,y,m=null,g=[],E=!1;this.setSize=function(S,x){o.setSize(S,x),a.setSize(S,x);for(let A=0;A<g.length;A++){let w=g[A];w.setSize&&w.setSize(S,x)}},this.setEffects=function(S){g=S,E=g.length>0&&g[0].isRenderPass===!0;let x=o.width,A=o.height;for(let w=0;w<g.length;w++){let I=g[w];I.setSize&&I.setSize(x,A)}},this.begin=function(S,x){if(p||S.toneMapping===Gn&&g.length===0)return!1;if(m=x,x!==null){let A=x.width,w=x.height;(o.width!==A||o.height!==w)&&this.setSize(A,w)}return E===!1&&S.setRenderTarget(o),y=S.toneMapping,S.toneMapping=Gn,!0},this.hasRenderPass=function(){return E},this.end=function(S,x){S.toneMapping=y,p=!0;let A=o,w=a;for(let I=0;I<g.length;I++){let v=g[I];if(v.enabled!==!1&&(v.render(S,w,A,x),v.needsSwap!==!1)){let R=A;A=w,w=R}}if(f!==S.outputColorSpace||d!==S.toneMapping){f=S.outputColorSpace,d=S.toneMapping,l.defines={},Qe.getTransfer(f)===lt&&(l.defines.SRGB_TRANSFER="");let I=R_[d];I&&(l.defines[I]=""),l.needsUpdate=!0}l.uniforms.tDiffuse.value=A.texture,S.setRenderTarget(m),S.render(h,u),m=null,p=!1},this.isCompositing=function(){return p},this.dispose=function(){o.depthTexture&&o.depthTexture.dispose(),o.dispose(),a.dispose(),c.dispose(),l.dispose()}}var fd=new Tt,ph=new xi(1,1),dd=new Vr,pd=new xa,md=new $r,Yf=[],Zf=[],Kf=new Float32Array(16),jf=new Float32Array(9),Jf=new Float32Array(4);function xr(i,e,t){let n=i[0];if(n<=0||n>0)return i;let s=e*t,r=Yf[s];if(r===void 0&&(r=new Float32Array(s),Yf[s]=r),e!==0){n.toArray(r,0);for(let o=1,a=0;o!==e;++o)a+=t,i[o].toArray(r,a)}return r}function kt(i,e){if(i.length!==e.length)return!1;for(let t=0,n=i.length;t<n;t++)if(i[t]!==e[t])return!1;return!0}function zt(i,e){for(let t=0,n=e.length;t<n;t++)i[t]=e[t]}function Lc(i,e){let t=Zf[e];t===void 0&&(t=new Int32Array(e),Zf[e]=t);for(let n=0;n!==e;++n)t[n]=i.allocateTextureUnit();return t}function P_(i,e){let t=this.cache;t[0]!==e&&(i.uniform1f(this.addr,e),t[0]=e)}function I_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2f(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(kt(t,e))return;i.uniform2fv(this.addr,e),zt(t,e)}}function L_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3f(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else if(e.r!==void 0)(t[0]!==e.r||t[1]!==e.g||t[2]!==e.b)&&(i.uniform3f(this.addr,e.r,e.g,e.b),t[0]=e.r,t[1]=e.g,t[2]=e.b);else{if(kt(t,e))return;i.uniform3fv(this.addr,e),zt(t,e)}}function D_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4f(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(kt(t,e))return;i.uniform4fv(this.addr,e),zt(t,e)}}function N_(i,e){let t=this.cache,n=e.elements;if(n===void 0){if(kt(t,e))return;i.uniformMatrix2fv(this.addr,!1,e),zt(t,e)}else{if(kt(t,n))return;Jf.set(n),i.uniformMatrix2fv(this.addr,!1,Jf),zt(t,n)}}function U_(i,e){let t=this.cache,n=e.elements;if(n===void 0){if(kt(t,e))return;i.uniformMatrix3fv(this.addr,!1,e),zt(t,e)}else{if(kt(t,n))return;jf.set(n),i.uniformMatrix3fv(this.addr,!1,jf),zt(t,n)}}function O_(i,e){let t=this.cache,n=e.elements;if(n===void 0){if(kt(t,e))return;i.uniformMatrix4fv(this.addr,!1,e),zt(t,e)}else{if(kt(t,n))return;Kf.set(n),i.uniformMatrix4fv(this.addr,!1,Kf),zt(t,n)}}function F_(i,e){let t=this.cache;t[0]!==e&&(i.uniform1i(this.addr,e),t[0]=e)}function B_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2i(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(kt(t,e))return;i.uniform2iv(this.addr,e),zt(t,e)}}function k_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3i(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else{if(kt(t,e))return;i.uniform3iv(this.addr,e),zt(t,e)}}function z_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4i(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(kt(t,e))return;i.uniform4iv(this.addr,e),zt(t,e)}}function H_(i,e){let t=this.cache;t[0]!==e&&(i.uniform1ui(this.addr,e),t[0]=e)}function V_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2ui(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(kt(t,e))return;i.uniform2uiv(this.addr,e),zt(t,e)}}function G_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3ui(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else{if(kt(t,e))return;i.uniform3uiv(this.addr,e),zt(t,e)}}function W_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4ui(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(kt(t,e))return;i.uniform4uiv(this.addr,e),zt(t,e)}}function X_(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s);let r;this.type===i.SAMPLER_2D_SHADOW?(ph.compareFunction=t.isReversedDepthBuffer()?wc:Ac,r=ph):r=fd,t.setTexture2D(e||r,s)}function q_(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s),t.setTexture3D(e||pd,s)}function Y_(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s),t.setTextureCube(e||md,s)}function Z_(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s),t.setTexture2DArray(e||dd,s)}function K_(i){switch(i){case 5126:return P_;case 35664:return I_;case 35665:return L_;case 35666:return D_;case 35674:return N_;case 35675:return U_;case 35676:return O_;case 5124:case 35670:return F_;case 35667:case 35671:return B_;case 35668:case 35672:return k_;case 35669:case 35673:return z_;case 5125:return H_;case 36294:return V_;case 36295:return G_;case 36296:return W_;case 35678:case 36198:case 36298:case 36306:case 35682:return X_;case 35679:case 36299:case 36307:return q_;case 35680:case 36300:case 36308:case 36293:return Y_;case 36289:case 36303:case 36311:case 36292:return Z_}}function j_(i,e){i.uniform1fv(this.addr,e)}function J_(i,e){let t=xr(e,this.size,2);i.uniform2fv(this.addr,t)}function $_(i,e){let t=xr(e,this.size,3);i.uniform3fv(this.addr,t)}function Q_(i,e){let t=xr(e,this.size,4);i.uniform4fv(this.addr,t)}function ex(i,e){let t=xr(e,this.size,4);i.uniformMatrix2fv(this.addr,!1,t)}function tx(i,e){let t=xr(e,this.size,9);i.uniformMatrix3fv(this.addr,!1,t)}function nx(i,e){let t=xr(e,this.size,16);i.uniformMatrix4fv(this.addr,!1,t)}function ix(i,e){i.uniform1iv(this.addr,e)}function sx(i,e){i.uniform2iv(this.addr,e)}function rx(i,e){i.uniform3iv(this.addr,e)}function ox(i,e){i.uniform4iv(this.addr,e)}function ax(i,e){i.uniform1uiv(this.addr,e)}function cx(i,e){i.uniform2uiv(this.addr,e)}function lx(i,e){i.uniform3uiv(this.addr,e)}function hx(i,e){i.uniform4uiv(this.addr,e)}function ux(i,e,t){let n=this.cache,s=e.length,r=Lc(t,s);kt(n,r)||(i.uniform1iv(this.addr,r),zt(n,r));let o;this.type===i.SAMPLER_2D_SHADOW?o=ph:o=fd;for(let a=0;a!==s;++a)t.setTexture2D(e[a]||o,r[a])}function fx(i,e,t){let n=this.cache,s=e.length,r=Lc(t,s);kt(n,r)||(i.uniform1iv(this.addr,r),zt(n,r));for(let o=0;o!==s;++o)t.setTexture3D(e[o]||pd,r[o])}function dx(i,e,t){let n=this.cache,s=e.length,r=Lc(t,s);kt(n,r)||(i.uniform1iv(this.addr,r),zt(n,r));for(let o=0;o!==s;++o)t.setTextureCube(e[o]||md,r[o])}function px(i,e,t){let n=this.cache,s=e.length,r=Lc(t,s);kt(n,r)||(i.uniform1iv(this.addr,r),zt(n,r));for(let o=0;o!==s;++o)t.setTexture2DArray(e[o]||dd,r[o])}function mx(i){switch(i){case 5126:return j_;case 35664:return J_;case 35665:return $_;case 35666:return Q_;case 35674:return ex;case 35675:return tx;case 35676:return nx;case 5124:case 35670:return ix;case 35667:case 35671:return sx;case 35668:case 35672:return rx;case 35669:case 35673:return ox;case 5125:return ax;case 36294:return cx;case 36295:return lx;case 36296:return hx;case 35678:case 36198:case 36298:case 36306:case 35682:return ux;case 35679:case 36299:case 36307:return fx;case 35680:case 36300:case 36308:case 36293:return dx;case 36289:case 36303:case 36311:case 36292:return px}}var mh=class{constructor(e,t,n){this.id=e,this.addr=n,this.cache=[],this.type=t.type,this.setValue=K_(t.type)}},gh=class{constructor(e,t,n){this.id=e,this.addr=n,this.cache=[],this.type=t.type,this.size=t.size,this.setValue=mx(t.type)}},_h=class{constructor(e){this.id=e,this.seq=[],this.map={}}setValue(e,t,n){let s=this.seq;for(let r=0,o=s.length;r!==o;++r){let a=s[r];a.setValue(e,t[a.id],n)}}},fh=/(\w+)(\])?(\[|\.)?/g;function $f(i,e){i.seq.push(e),i.map[e.id]=e}function gx(i,e,t){let n=i.name,s=n.length;for(fh.lastIndex=0;;){let r=fh.exec(n),o=fh.lastIndex,a=r[1],c=r[2]==="]",l=r[3];if(c&&(a=a|0),l===void 0||l==="["&&o+2===s){$f(t,l===void 0?new mh(a,i,e):new gh(a,i,e));break}else{let u=t.map[a];u===void 0&&(u=new _h(a),$f(t,u)),t=u}}}var gr=class{constructor(e,t){this.seq=[],this.map={};let n=e.getProgramParameter(t,e.ACTIVE_UNIFORMS);for(let o=0;o<n;++o){let a=e.getActiveUniform(t,o),c=e.getUniformLocation(t,a.name);gx(a,c,this)}let s=[],r=[];for(let o of this.seq)o.type===e.SAMPLER_2D_SHADOW||o.type===e.SAMPLER_CUBE_SHADOW||o.type===e.SAMPLER_2D_ARRAY_SHADOW?s.push(o):r.push(o);s.length>0&&(this.seq=s.concat(r))}setValue(e,t,n,s){let r=this.map[t];r!==void 0&&r.setValue(e,n,s)}setOptional(e,t,n){let s=t[n];s!==void 0&&this.setValue(e,n,s)}static upload(e,t,n,s){for(let r=0,o=t.length;r!==o;++r){let a=t[r],c=n[a.id];c.needsUpdate!==!1&&a.setValue(e,c.value,s)}}static seqWithValue(e,t){let n=[];for(let s=0,r=e.length;s!==r;++s){let o=e[s];o.id in t&&n.push(o)}return n}};function Qf(i,e,t){let n=i.createShader(e);return i.shaderSource(n,t),i.compileShader(n),n}var _x=37297,xx=0;function yx(i,e){let t=i.split(`
`),n=[],s=Math.max(e-6,0),r=Math.min(e+6,t.length);for(let o=s;o<r;o++){let a=o+1;n.push(`${a===e?">":" "} ${a}: ${t[o]}`)}return n.join(`
`)}var ed=new Ze;function vx(i){Qe._getMatrix(ed,Qe.workingColorSpace,i);let e=`mat3( ${ed.elements.map(t=>t.toFixed(4))} )`;switch(Qe.getTransfer(i)){case zr:return[e,"LinearTransferOETF"];case lt:return[e,"sRGBTransferOETF"];default:return ke("WebGLProgram: Unsupported color space: ",i),[e,"LinearTransferOETF"]}}function td(i,e,t){let n=i.getShaderParameter(e,i.COMPILE_STATUS),r=(i.getShaderInfoLog(e)||"").trim();if(n&&r==="")return"";let o=/ERROR: 0:(\d+)/.exec(r);if(o){let a=parseInt(o[1]);return t.toUpperCase()+`

`+r+`

`+yx(i.getShaderSource(e),a)}else return r}function bx(i,e){let t=vx(e);return[`vec4 ${i}( vec4 value ) {`,`	return ${t[1]}( vec4( value.rgb * ${t[0]}, value.a ) );`,"}"].join(`
`)}var Mx={[Fl]:"Linear",[Bl]:"Reinhard",[kl]:"Cineon",[bs]:"ACESFilmic",[Hl]:"AgX",[Vl]:"Neutral",[zl]:"Custom"};function Sx(i,e){let t=Mx[e];return t===void 0?(ke("WebGLProgram: Unsupported toneMapping:",e),"vec3 "+i+"( vec3 color ) { return LinearToneMapping( color ); }"):"vec3 "+i+"( vec3 color ) { return "+t+"ToneMapping( color ); }"}var Cc=new B;function Tx(){Qe.getLuminanceCoefficients(Cc);let i=Cc.x.toFixed(4),e=Cc.y.toFixed(4),t=Cc.z.toFixed(4);return["float luminance( const in vec3 rgb ) {",`	const vec3 weights = vec3( ${i}, ${e}, ${t} );`,"	return dot( weights, rgb );","}"].join(`
`)}function Ex(i){return[i.extensionClipCullDistance?"#extension GL_ANGLE_clip_cull_distance : require":"",i.extensionMultiDraw?"#extension GL_ANGLE_multi_draw : require":""].filter(Ao).join(`
`)}function Ax(i){let e=[];for(let t in i){let n=i[t];n!==!1&&e.push("#define "+t+" "+n)}return e.join(`
`)}function wx(i,e){let t={},n=i.getProgramParameter(e,i.ACTIVE_ATTRIBUTES);for(let s=0;s<n;s++){let r=i.getActiveAttrib(e,s),o=r.name,a=1;r.type===i.FLOAT_MAT2&&(a=2),r.type===i.FLOAT_MAT3&&(a=3),r.type===i.FLOAT_MAT4&&(a=4),t[o]={type:r.type,location:i.getAttribLocation(e,o),locationSize:a}}return t}function Ao(i){return i!==""}function nd(i,e){let t=e.numSpotLightShadows+e.numSpotLightMaps-e.numSpotLightShadowsWithMaps;return i.replace(/NUM_DIR_LIGHTS/g,e.numDirLights).replace(/NUM_SPOT_LIGHTS/g,e.numSpotLights).replace(/NUM_SPOT_LIGHT_MAPS/g,e.numSpotLightMaps).replace(/NUM_SPOT_LIGHT_COORDS/g,t).replace(/NUM_RECT_AREA_LIGHTS/g,e.numRectAreaLights).replace(/NUM_POINT_LIGHTS/g,e.numPointLights).replace(/NUM_HEMI_LIGHTS/g,e.numHemiLights).replace(/NUM_DIR_LIGHT_SHADOWS/g,e.numDirLightShadows).replace(/NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS/g,e.numSpotLightShadowsWithMaps).replace(/NUM_SPOT_LIGHT_SHADOWS/g,e.numSpotLightShadows).replace(/NUM_POINT_LIGHT_SHADOWS/g,e.numPointLightShadows)}function id(i,e){return i.replace(/NUM_CLIPPING_PLANES/g,e.numClippingPlanes).replace(/UNION_CLIPPING_PLANES/g,e.numClippingPlanes-e.numClipIntersection)}var Rx=/^[ \t]*#include +<([\w\d./]+)>/gm;function xh(i){return i.replace(Rx,Px)}var Cx=new Map;function Px(i,e){let t=tt[e];if(t===void 0){let n=Cx.get(e);if(n!==void 0)t=tt[n],ke('WebGLRenderer: Shader chunk "%s" has been deprecated. Use "%s" instead.',e,n);else throw new Error("THREE.WebGLProgram: Can not resolve #include <"+e+">")}return xh(t)}var Ix=/#pragma unroll_loop_start\s+for\s*\(\s*int\s+i\s*=\s*(\d+)\s*;\s*i\s*<\s*(\d+)\s*;\s*i\s*\+\+\s*\)\s*{([\s\S]+?)}\s+#pragma unroll_loop_end/g;function sd(i){return i.replace(Ix,Lx)}function Lx(i,e,t,n){let s="";for(let r=parseInt(e);r<parseInt(t);r++)s+=n.replace(/\[\s*i\s*\]/g,"[ "+r+" ]").replace(/UNROLLED_LOOP_INDEX/g,r);return s}function rd(i){let e=`precision ${i.precision} float;
	precision ${i.precision} int;
	precision ${i.precision} sampler2D;
	precision ${i.precision} samplerCube;
	precision ${i.precision} sampler3D;
	precision ${i.precision} sampler2DArray;
	precision ${i.precision} sampler2DShadow;
	precision ${i.precision} samplerCubeShadow;
	precision ${i.precision} sampler2DArrayShadow;
	precision ${i.precision} isampler2D;
	precision ${i.precision} isampler3D;
	precision ${i.precision} isamplerCube;
	precision ${i.precision} isampler2DArray;
	precision ${i.precision} usampler2D;
	precision ${i.precision} usampler3D;
	precision ${i.precision} usamplerCube;
	precision ${i.precision} usampler2DArray;
	`;return i.precision==="highp"?e+=`
#define HIGH_PRECISION`:i.precision==="mediump"?e+=`
#define MEDIUM_PRECISION`:i.precision==="lowp"&&(e+=`
#define LOW_PRECISION`),e}var Dx={[mo]:"SHADOWMAP_TYPE_PCF",[ar]:"SHADOWMAP_TYPE_VSM"};function Nx(i){return Dx[i.shadowMapType]||"SHADOWMAP_TYPE_BASIC"}var Ux={[Ki]:"ENVMAP_TYPE_CUBE",[Ms]:"ENVMAP_TYPE_CUBE",[go]:"ENVMAP_TYPE_CUBE_UV"};function Ox(i){return i.envMap===!1?"ENVMAP_TYPE_CUBE":Ux[i.envMapMode]||"ENVMAP_TYPE_CUBE"}var Fx={[Ms]:"ENVMAP_MODE_REFRACTION"};function Bx(i){return i.envMap===!1?"ENVMAP_MODE_REFLECTION":Fx[i.envMapMode]||"ENVMAP_MODE_REFLECTION"}var kx={[Ol]:"ENVMAP_BLENDING_MULTIPLY",[yf]:"ENVMAP_BLENDING_MIX",[vf]:"ENVMAP_BLENDING_ADD"};function zx(i){return i.envMap===!1?"ENVMAP_BLENDING_NONE":kx[i.combine]||"ENVMAP_BLENDING_NONE"}function Hx(i){let e=i.envMapCubeUVHeight;if(e===null)return null;let t=Math.log2(e)-2,n=1/e;return{texelWidth:1/(3*Math.max(Math.pow(2,t),112)),texelHeight:n,maxMip:t}}function Vx(i,e,t,n){let s=i.getContext(),r=t.defines,o=t.vertexShader,a=t.fragmentShader,c=Nx(t),l=Ox(t),h=Bx(t),u=zx(t),f=Hx(t),d=Ex(t),p=Ax(r),y=s.createProgram(),m,g,E=t.glslVersion?"#version "+t.glslVersion+`
`:"";t.isRawShaderMaterial?(m=["#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,p].filter(Ao).join(`
`),m.length>0&&(m+=`
`),g=["#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,p].filter(Ao).join(`
`),g.length>0&&(g+=`
`)):(m=[rd(t),"#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,p,t.extensionClipCullDistance?"#define USE_CLIP_DISTANCE":"",t.batching?"#define USE_BATCHING":"",t.batchingColor?"#define USE_BATCHING_COLOR":"",t.instancing?"#define USE_INSTANCING":"",t.instancingColor?"#define USE_INSTANCING_COLOR":"",t.instancingMorph?"#define USE_INSTANCING_MORPH":"",t.useFog&&t.fog?"#define USE_FOG":"",t.useFog&&t.fogExp2?"#define FOG_EXP2":"",t.map?"#define USE_MAP":"",t.envMap?"#define USE_ENVMAP":"",t.envMap?"#define "+h:"",t.lightMap?"#define USE_LIGHTMAP":"",t.aoMap?"#define USE_AOMAP":"",t.bumpMap?"#define USE_BUMPMAP":"",t.normalMap?"#define USE_NORMALMAP":"",t.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",t.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",t.displacementMap?"#define USE_DISPLACEMENTMAP":"",t.emissiveMap?"#define USE_EMISSIVEMAP":"",t.anisotropy?"#define USE_ANISOTROPY":"",t.anisotropyMap?"#define USE_ANISOTROPYMAP":"",t.clearcoatMap?"#define USE_CLEARCOATMAP":"",t.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",t.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",t.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",t.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",t.specularMap?"#define USE_SPECULARMAP":"",t.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",t.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",t.roughnessMap?"#define USE_ROUGHNESSMAP":"",t.metalnessMap?"#define USE_METALNESSMAP":"",t.alphaMap?"#define USE_ALPHAMAP":"",t.alphaHash?"#define USE_ALPHAHASH":"",t.transmission?"#define USE_TRANSMISSION":"",t.transmissionMap?"#define USE_TRANSMISSIONMAP":"",t.thicknessMap?"#define USE_THICKNESSMAP":"",t.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",t.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",t.mapUv?"#define MAP_UV "+t.mapUv:"",t.alphaMapUv?"#define ALPHAMAP_UV "+t.alphaMapUv:"",t.lightMapUv?"#define LIGHTMAP_UV "+t.lightMapUv:"",t.aoMapUv?"#define AOMAP_UV "+t.aoMapUv:"",t.emissiveMapUv?"#define EMISSIVEMAP_UV "+t.emissiveMapUv:"",t.bumpMapUv?"#define BUMPMAP_UV "+t.bumpMapUv:"",t.normalMapUv?"#define NORMALMAP_UV "+t.normalMapUv:"",t.displacementMapUv?"#define DISPLACEMENTMAP_UV "+t.displacementMapUv:"",t.metalnessMapUv?"#define METALNESSMAP_UV "+t.metalnessMapUv:"",t.roughnessMapUv?"#define ROUGHNESSMAP_UV "+t.roughnessMapUv:"",t.anisotropyMapUv?"#define ANISOTROPYMAP_UV "+t.anisotropyMapUv:"",t.clearcoatMapUv?"#define CLEARCOATMAP_UV "+t.clearcoatMapUv:"",t.clearcoatNormalMapUv?"#define CLEARCOAT_NORMALMAP_UV "+t.clearcoatNormalMapUv:"",t.clearcoatRoughnessMapUv?"#define CLEARCOAT_ROUGHNESSMAP_UV "+t.clearcoatRoughnessMapUv:"",t.iridescenceMapUv?"#define IRIDESCENCEMAP_UV "+t.iridescenceMapUv:"",t.iridescenceThicknessMapUv?"#define IRIDESCENCE_THICKNESSMAP_UV "+t.iridescenceThicknessMapUv:"",t.sheenColorMapUv?"#define SHEEN_COLORMAP_UV "+t.sheenColorMapUv:"",t.sheenRoughnessMapUv?"#define SHEEN_ROUGHNESSMAP_UV "+t.sheenRoughnessMapUv:"",t.specularMapUv?"#define SPECULARMAP_UV "+t.specularMapUv:"",t.specularColorMapUv?"#define SPECULAR_COLORMAP_UV "+t.specularColorMapUv:"",t.specularIntensityMapUv?"#define SPECULAR_INTENSITYMAP_UV "+t.specularIntensityMapUv:"",t.transmissionMapUv?"#define TRANSMISSIONMAP_UV "+t.transmissionMapUv:"",t.thicknessMapUv?"#define THICKNESSMAP_UV "+t.thicknessMapUv:"",t.vertexTangents&&t.flatShading===!1?"#define USE_TANGENT":"",t.vertexNormals?"#define HAS_NORMAL":"",t.vertexColors?"#define USE_COLOR":"",t.vertexAlphas?"#define USE_COLOR_ALPHA":"",t.vertexUv1s?"#define USE_UV1":"",t.vertexUv2s?"#define USE_UV2":"",t.vertexUv3s?"#define USE_UV3":"",t.pointsUvs?"#define USE_POINTS_UV":"",t.flatShading?"#define FLAT_SHADED":"",t.skinning?"#define USE_SKINNING":"",t.morphTargets?"#define USE_MORPHTARGETS":"",t.morphNormals&&t.flatShading===!1?"#define USE_MORPHNORMALS":"",t.morphColors?"#define USE_MORPHCOLORS":"",t.morphTargetsCount>0?"#define MORPHTARGETS_TEXTURE_STRIDE "+t.morphTextureStride:"",t.morphTargetsCount>0?"#define MORPHTARGETS_COUNT "+t.morphTargetsCount:"",t.doubleSided?"#define DOUBLE_SIDED":"",t.flipSided?"#define FLIP_SIDED":"",t.shadowMapEnabled?"#define USE_SHADOWMAP":"",t.shadowMapEnabled?"#define "+c:"",t.sizeAttenuation?"#define USE_SIZEATTENUATION":"",t.numLightProbes>0?"#define USE_LIGHT_PROBES":"",t.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",t.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 modelMatrix;","uniform mat4 modelViewMatrix;","uniform mat4 projectionMatrix;","uniform mat4 viewMatrix;","uniform mat3 normalMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;","#ifdef USE_INSTANCING","	attribute mat4 instanceMatrix;","#endif","#ifdef USE_INSTANCING_COLOR","	attribute vec3 instanceColor;","#endif","#ifdef USE_INSTANCING_MORPH","	uniform sampler2D morphTexture;","#endif","attribute vec3 position;","attribute vec3 normal;","attribute vec2 uv;","#ifdef USE_UV1","	attribute vec2 uv1;","#endif","#ifdef USE_UV2","	attribute vec2 uv2;","#endif","#ifdef USE_UV3","	attribute vec2 uv3;","#endif","#ifdef USE_TANGENT","	attribute vec4 tangent;","#endif","#if defined( USE_COLOR_ALPHA )","	attribute vec4 color;","#elif defined( USE_COLOR )","	attribute vec3 color;","#endif","#ifdef USE_SKINNING","	attribute vec4 skinIndex;","	attribute vec4 skinWeight;","#endif",`
`].filter(Ao).join(`
`),g=[rd(t),"#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,p,t.useFog&&t.fog?"#define USE_FOG":"",t.useFog&&t.fogExp2?"#define FOG_EXP2":"",t.alphaToCoverage?"#define ALPHA_TO_COVERAGE":"",t.map?"#define USE_MAP":"",t.matcap?"#define USE_MATCAP":"",t.envMap?"#define USE_ENVMAP":"",t.envMap?"#define "+l:"",t.envMap?"#define "+h:"",t.envMap?"#define "+u:"",f?"#define CUBEUV_TEXEL_WIDTH "+f.texelWidth:"",f?"#define CUBEUV_TEXEL_HEIGHT "+f.texelHeight:"",f?"#define CUBEUV_MAX_MIP "+f.maxMip+".0":"",t.lightMap?"#define USE_LIGHTMAP":"",t.aoMap?"#define USE_AOMAP":"",t.bumpMap?"#define USE_BUMPMAP":"",t.normalMap?"#define USE_NORMALMAP":"",t.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",t.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",t.packedNormalMap?"#define USE_PACKED_NORMALMAP":"",t.emissiveMap?"#define USE_EMISSIVEMAP":"",t.anisotropy?"#define USE_ANISOTROPY":"",t.anisotropyMap?"#define USE_ANISOTROPYMAP":"",t.clearcoat?"#define USE_CLEARCOAT":"",t.clearcoatMap?"#define USE_CLEARCOATMAP":"",t.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",t.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",t.dispersion?"#define USE_DISPERSION":"",t.iridescence?"#define USE_IRIDESCENCE":"",t.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",t.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",t.specularMap?"#define USE_SPECULARMAP":"",t.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",t.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",t.roughnessMap?"#define USE_ROUGHNESSMAP":"",t.metalnessMap?"#define USE_METALNESSMAP":"",t.alphaMap?"#define USE_ALPHAMAP":"",t.alphaTest?"#define USE_ALPHATEST":"",t.alphaHash?"#define USE_ALPHAHASH":"",t.sheen?"#define USE_SHEEN":"",t.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",t.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",t.transmission?"#define USE_TRANSMISSION":"",t.transmissionMap?"#define USE_TRANSMISSIONMAP":"",t.thicknessMap?"#define USE_THICKNESSMAP":"",t.vertexTangents&&t.flatShading===!1?"#define USE_TANGENT":"",t.vertexColors||t.instancingColor?"#define USE_COLOR":"",t.vertexAlphas||t.batchingColor?"#define USE_COLOR_ALPHA":"",t.vertexUv1s?"#define USE_UV1":"",t.vertexUv2s?"#define USE_UV2":"",t.vertexUv3s?"#define USE_UV3":"",t.pointsUvs?"#define USE_POINTS_UV":"",t.gradientMap?"#define USE_GRADIENTMAP":"",t.flatShading?"#define FLAT_SHADED":"",t.doubleSided?"#define DOUBLE_SIDED":"",t.flipSided?"#define FLIP_SIDED":"",t.shadowMapEnabled?"#define USE_SHADOWMAP":"",t.shadowMapEnabled?"#define "+c:"",t.premultipliedAlpha?"#define PREMULTIPLIED_ALPHA":"",t.numLightProbes>0?"#define USE_LIGHT_PROBES":"",t.numLightProbeGrids>0?"#define USE_LIGHT_PROBES_GRID":"",t.decodeVideoTexture?"#define DECODE_VIDEO_TEXTURE":"",t.decodeVideoTextureEmissive?"#define DECODE_VIDEO_TEXTURE_EMISSIVE":"",t.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",t.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 viewMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;",t.toneMapping!==Gn?"#define TONE_MAPPING":"",t.toneMapping!==Gn?tt.tonemapping_pars_fragment:"",t.toneMapping!==Gn?Sx("toneMapping",t.toneMapping):"",t.dithering?"#define DITHERING":"",t.opaque?"#define OPAQUE":"",tt.colorspace_pars_fragment,bx("linearToOutputTexel",t.outputColorSpace),Tx(),t.useDepthPacking?"#define DEPTH_PACKING "+t.depthPacking:"",`
`].filter(Ao).join(`
`)),o=xh(o),o=nd(o,t),o=id(o,t),a=xh(a),a=nd(a,t),a=id(a,t),o=sd(o),a=sd(a),t.isRawShaderMaterial!==!0&&(E=`#version 300 es
`,m=[d,"#define attribute in","#define varying out","#define texture2D texture"].join(`
`)+`
`+m,g=["#define varying in",t.glslVersion===pr?"":"layout(location = 0) out highp vec4 pc_fragColor;",t.glslVersion===pr?"":"#define gl_FragColor pc_fragColor","#define gl_FragDepthEXT gl_FragDepth","#define texture2D texture","#define textureCube texture","#define texture2DProj textureProj","#define texture2DLodEXT textureLod","#define texture2DProjLodEXT textureProjLod","#define textureCubeLodEXT textureLod","#define texture2DGradEXT textureGrad","#define texture2DProjGradEXT textureProjGrad","#define textureCubeGradEXT textureGrad"].join(`
`)+`
`+g);let S=E+m+o,x=E+g+a,A=Qf(s,s.VERTEX_SHADER,S),w=Qf(s,s.FRAGMENT_SHADER,x);s.attachShader(y,A),s.attachShader(y,w),t.index0AttributeName!==void 0?s.bindAttribLocation(y,0,t.index0AttributeName):t.hasPositionAttribute===!0&&s.bindAttribLocation(y,0,"position"),s.linkProgram(y);function I(N){if(i.debug.checkShaderErrors){let H=s.getProgramInfoLog(y)||"",oe=s.getShaderInfoLog(A)||"",ie=s.getShaderInfoLog(w)||"",X=H.trim(),ee=oe.trim(),Y=ie.trim(),G=!0,ge=!0;if(s.getProgramParameter(y,s.LINK_STATUS)===!1)if(G=!1,typeof i.debug.onShaderError=="function")i.debug.onShaderError(s,y,A,w);else{let ye=td(s,A,"vertex"),Me=td(s,w,"fragment");Ke("WebGLProgram: Shader Error "+s.getError()+" - VALIDATE_STATUS "+s.getProgramParameter(y,s.VALIDATE_STATUS)+`

Material Name: `+N.name+`
Material Type: `+N.type+`

Program Info Log: `+X+`
`+ye+`
`+Me)}else X!==""?ke("WebGLProgram: Program Info Log:",X):(ee===""||Y==="")&&(ge=!1);ge&&(N.diagnostics={runnable:G,programLog:X,vertexShader:{log:ee,prefix:m},fragmentShader:{log:Y,prefix:g}})}s.deleteShader(A),s.deleteShader(w),v=new gr(s,y),R=wx(s,y)}let v;this.getUniforms=function(){return v===void 0&&I(this),v};let R;this.getAttributes=function(){return R===void 0&&I(this),R};let F=t.rendererExtensionParallelShaderCompile===!1;return this.isReady=function(){return F===!1&&(F=s.getProgramParameter(y,_x)),F},this.destroy=function(){n.releaseStatesOfProgram(this),s.deleteProgram(y),this.program=void 0},this.type=t.shaderType,this.name=t.shaderName,this.id=xx++,this.cacheKey=e,this.usedTimes=1,this.program=y,this.vertexShader=A,this.fragmentShader=w,this}var Gx=0,yh=class{constructor(){this.shaderCache=new Map,this.materialCache=new Map}update(e,t,n){let s=this._getShaderCacheForMaterial(e);return s.has(t)===!1&&(s.add(t),t.usedTimes++),s.has(n)===!1&&(s.add(n),n.usedTimes++),this}remove(e){let t=this.materialCache.get(e);for(let n of t)n.usedTimes--,n.usedTimes===0&&this.shaderCache.delete(n.code);return this.materialCache.delete(e),this}getVertexShaderStage(e){return this._getShaderStage(e.vertexShader)}getFragmentShaderStage(e){return this._getShaderStage(e.fragmentShader)}dispose(){this.shaderCache.clear(),this.materialCache.clear()}_getShaderCacheForMaterial(e){let t=this.materialCache,n=t.get(e);return n===void 0&&(n=new Set,t.set(e,n)),n}_getShaderStage(e){let t=this.shaderCache,n=t.get(e);return n===void 0&&(n=new vh(e),t.set(e,n)),n}},vh=class{constructor(e){this.id=Gx++,this.code=e,this.usedTimes=0}};function Wx(i){return i===Ji||i===bo||i===Mo}function Xx(i,e,t,n,s,r){let o=new Gr,a=new yh,c=new Set,l=[],h=new Map,u=n.logarithmicDepthBuffer,f=n.precision,d={MeshDepthMaterial:"depth",MeshDistanceMaterial:"distance",MeshNormalMaterial:"normal",MeshBasicMaterial:"basic",MeshLambertMaterial:"lambert",MeshPhongMaterial:"phong",MeshToonMaterial:"toon",MeshStandardMaterial:"physical",MeshPhysicalMaterial:"physical",MeshMatcapMaterial:"matcap",LineBasicMaterial:"basic",LineDashedMaterial:"dashed",PointsMaterial:"points",ShadowMaterial:"shadow",SpriteMaterial:"sprite"};function p(v){return c.add(v),v===0?"uv":`uv${v}`}function y(v,R,F,N,H,oe){let ie=N.fog,X=H.geometry,ee=v.isMeshStandardMaterial||v.isMeshLambertMaterial||v.isMeshPhongMaterial?N.environment:null,Y=v.isMeshStandardMaterial||v.isMeshLambertMaterial&&!v.envMap||v.isMeshPhongMaterial&&!v.envMap,G=e.get(v.envMap||ee,Y),ge=G&&G.mapping===go?G.image.height:null,ye=d[v.type];v.precision!==null&&(f=n.getMaxPrecision(v.precision),f!==v.precision&&ke("WebGLProgram.getParameters:",v.precision,"not supported, using",f,"instead."));let Me=X.morphAttributes.position||X.morphAttributes.normal||X.morphAttributes.color,Re=Me!==void 0?Me.length:0,ze=0;X.morphAttributes.position!==void 0&&(ze=1),X.morphAttributes.normal!==void 0&&(ze=2),X.morphAttributes.color!==void 0&&(ze=3);let qe,We,pe,q;if(ye){let Be=si[ye];qe=Be.vertexShader,We=Be.fragmentShader}else{qe=v.vertexShader,We=v.fragmentShader;let Be=a.getVertexShaderStage(v),dt=a.getFragmentShaderStage(v);a.update(v,Be,dt),pe=Be.id,q=dt.id}let U=i.getRenderTarget(),L=i.state.buffers.depth.getReversed(),P=H.isInstancedMesh===!0,z=H.isBatchedMesh===!0,he=!!v.map,me=!!v.matcap,C=!!G,V=!!v.aoMap,D=!!v.lightMap,j=!!v.bumpMap&&v.wireframe===!1,$=!!v.normalMap,le=!!v.displacementMap,te=!!v.emissiveMap,ne=!!v.metalnessMap,k=!!v.roughnessMap,b=v.anisotropy>0,we=v.clearcoat>0,Ae=v.dispersion>0,T=v.iridescence>0,_=v.sheen>0,O=v.transmission>0,W=b&&!!v.anisotropyMap,Q=we&&!!v.clearcoatMap,_e=we&&!!v.clearcoatNormalMap,Se=we&&!!v.clearcoatRoughnessMap,ae=T&&!!v.iridescenceMap,fe=T&&!!v.iridescenceThicknessMap,J=_&&!!v.sheenColorMap,xe=_&&!!v.sheenRoughnessMap,ue=!!v.specularMap,ve=!!v.specularColorMap,Ee=!!v.specularIntensityMap,De=O&&!!v.transmissionMap,je=O&&!!v.thicknessMap,Z=!!v.gradientMap,Pe=!!v.alphaMap,be=v.alphaTest>0,Ie=!!v.alphaHash,Ce=!!v.extensions,Te=Gn;v.toneMapped&&(U===null||U.isXRRenderTarget===!0)&&(Te=i.toneMapping);let He={shaderID:ye,shaderType:v.type,shaderName:v.name,vertexShader:qe,fragmentShader:We,defines:v.defines,customVertexShaderID:pe,customFragmentShaderID:q,isRawShaderMaterial:v.isRawShaderMaterial===!0,glslVersion:v.glslVersion,precision:f,batching:z,batchingColor:z&&H._colorsTexture!==null,instancing:P,instancingColor:P&&H.instanceColor!==null,instancingMorph:P&&H.morphTexture!==null,outputColorSpace:U===null?i.outputColorSpace:U.isXRRenderTarget===!0?U.texture.colorSpace:Qe.workingColorSpace,alphaToCoverage:!!v.alphaToCoverage,map:he,matcap:me,envMap:C,envMapMode:C&&G.mapping,envMapCubeUVHeight:ge,aoMap:V,lightMap:D,bumpMap:j,normalMap:$,displacementMap:le,emissiveMap:te,normalMapObjectSpace:$&&v.normalMapType===Tf,normalMapTangentSpace:$&&v.normalMapType===Ec,packedNormalMap:$&&v.normalMapType===Ec&&Wx(v.normalMap.format),metalnessMap:ne,roughnessMap:k,anisotropy:b,anisotropyMap:W,clearcoat:we,clearcoatMap:Q,clearcoatNormalMap:_e,clearcoatRoughnessMap:Se,dispersion:Ae,iridescence:T,iridescenceMap:ae,iridescenceThicknessMap:fe,sheen:_,sheenColorMap:J,sheenRoughnessMap:xe,specularMap:ue,specularColorMap:ve,specularIntensityMap:Ee,transmission:O,transmissionMap:De,thicknessMap:je,gradientMap:Z,opaque:v.transparent===!1&&v.blending===hs&&v.alphaToCoverage===!1,alphaMap:Pe,alphaTest:be,alphaHash:Ie,combine:v.combine,mapUv:he&&p(v.map.channel),aoMapUv:V&&p(v.aoMap.channel),lightMapUv:D&&p(v.lightMap.channel),bumpMapUv:j&&p(v.bumpMap.channel),normalMapUv:$&&p(v.normalMap.channel),displacementMapUv:le&&p(v.displacementMap.channel),emissiveMapUv:te&&p(v.emissiveMap.channel),metalnessMapUv:ne&&p(v.metalnessMap.channel),roughnessMapUv:k&&p(v.roughnessMap.channel),anisotropyMapUv:W&&p(v.anisotropyMap.channel),clearcoatMapUv:Q&&p(v.clearcoatMap.channel),clearcoatNormalMapUv:_e&&p(v.clearcoatNormalMap.channel),clearcoatRoughnessMapUv:Se&&p(v.clearcoatRoughnessMap.channel),iridescenceMapUv:ae&&p(v.iridescenceMap.channel),iridescenceThicknessMapUv:fe&&p(v.iridescenceThicknessMap.channel),sheenColorMapUv:J&&p(v.sheenColorMap.channel),sheenRoughnessMapUv:xe&&p(v.sheenRoughnessMap.channel),specularMapUv:ue&&p(v.specularMap.channel),specularColorMapUv:ve&&p(v.specularColorMap.channel),specularIntensityMapUv:Ee&&p(v.specularIntensityMap.channel),transmissionMapUv:De&&p(v.transmissionMap.channel),thicknessMapUv:je&&p(v.thicknessMap.channel),alphaMapUv:Pe&&p(v.alphaMap.channel),vertexTangents:!!X.attributes.tangent&&($||b),vertexNormals:!!X.attributes.normal,vertexColors:v.vertexColors,vertexAlphas:v.vertexColors===!0&&!!X.attributes.color&&X.attributes.color.itemSize===4,pointsUvs:H.isPoints===!0&&!!X.attributes.uv&&(he||Pe),fog:!!ie,useFog:v.fog===!0,fogExp2:!!ie&&ie.isFogExp2,flatShading:v.wireframe===!1&&(v.flatShading===!0||X.attributes.normal===void 0&&$===!1&&(v.isMeshLambertMaterial||v.isMeshPhongMaterial||v.isMeshStandardMaterial||v.isMeshPhysicalMaterial)),sizeAttenuation:v.sizeAttenuation===!0,logarithmicDepthBuffer:u,reversedDepthBuffer:L,skinning:H.isSkinnedMesh===!0,hasPositionAttribute:X.attributes.position!==void 0,morphTargets:X.morphAttributes.position!==void 0,morphNormals:X.morphAttributes.normal!==void 0,morphColors:X.morphAttributes.color!==void 0,morphTargetsCount:Re,morphTextureStride:ze,numDirLights:R.directional.length,numPointLights:R.point.length,numSpotLights:R.spot.length,numSpotLightMaps:R.spotLightMap.length,numRectAreaLights:R.rectArea.length,numHemiLights:R.hemi.length,numDirLightShadows:R.directionalShadowMap.length,numPointLightShadows:R.pointShadowMap.length,numSpotLightShadows:R.spotShadowMap.length,numSpotLightShadowsWithMaps:R.numSpotLightShadowsWithMaps,numLightProbes:R.numLightProbes,numLightProbeGrids:oe.length,numClippingPlanes:r.numPlanes,numClipIntersection:r.numIntersection,dithering:v.dithering,shadowMapEnabled:i.shadowMap.enabled&&F.length>0,shadowMapType:i.shadowMap.type,toneMapping:Te,decodeVideoTexture:he&&v.map.isVideoTexture===!0&&Qe.getTransfer(v.map.colorSpace)===lt,decodeVideoTextureEmissive:te&&v.emissiveMap.isVideoTexture===!0&&Qe.getTransfer(v.emissiveMap.colorSpace)===lt,premultipliedAlpha:v.premultipliedAlpha,doubleSided:v.side===Bt,flipSided:v.side===Ft,useDepthPacking:v.depthPacking>=0,depthPacking:v.depthPacking||0,index0AttributeName:v.index0AttributeName,extensionClipCullDistance:Ce&&v.extensions.clipCullDistance===!0&&t.has("WEBGL_clip_cull_distance"),extensionMultiDraw:(Ce&&v.extensions.multiDraw===!0||z)&&t.has("WEBGL_multi_draw"),rendererExtensionParallelShaderCompile:t.has("KHR_parallel_shader_compile"),customProgramCacheKey:v.customProgramCacheKey()};return He.vertexUv1s=c.has(1),He.vertexUv2s=c.has(2),He.vertexUv3s=c.has(3),c.clear(),He}function m(v){let R=[];if(v.shaderID?R.push(v.shaderID):(R.push(v.customVertexShaderID),R.push(v.customFragmentShaderID)),v.defines!==void 0)for(let F in v.defines)R.push(F),R.push(v.defines[F]);return v.isRawShaderMaterial===!1&&(g(R,v),E(R,v),R.push(i.outputColorSpace)),R.push(v.customProgramCacheKey),R.join()}function g(v,R){v.push(R.precision),v.push(R.outputColorSpace),v.push(R.envMapMode),v.push(R.envMapCubeUVHeight),v.push(R.mapUv),v.push(R.alphaMapUv),v.push(R.lightMapUv),v.push(R.aoMapUv),v.push(R.bumpMapUv),v.push(R.normalMapUv),v.push(R.displacementMapUv),v.push(R.emissiveMapUv),v.push(R.metalnessMapUv),v.push(R.roughnessMapUv),v.push(R.anisotropyMapUv),v.push(R.clearcoatMapUv),v.push(R.clearcoatNormalMapUv),v.push(R.clearcoatRoughnessMapUv),v.push(R.iridescenceMapUv),v.push(R.iridescenceThicknessMapUv),v.push(R.sheenColorMapUv),v.push(R.sheenRoughnessMapUv),v.push(R.specularMapUv),v.push(R.specularColorMapUv),v.push(R.specularIntensityMapUv),v.push(R.transmissionMapUv),v.push(R.thicknessMapUv),v.push(R.combine),v.push(R.fogExp2),v.push(R.sizeAttenuation),v.push(R.morphTargetsCount),v.push(R.morphAttributeCount),v.push(R.numDirLights),v.push(R.numPointLights),v.push(R.numSpotLights),v.push(R.numSpotLightMaps),v.push(R.numHemiLights),v.push(R.numRectAreaLights),v.push(R.numDirLightShadows),v.push(R.numPointLightShadows),v.push(R.numSpotLightShadows),v.push(R.numSpotLightShadowsWithMaps),v.push(R.numLightProbes),v.push(R.shadowMapType),v.push(R.toneMapping),v.push(R.numClippingPlanes),v.push(R.numClipIntersection),v.push(R.depthPacking)}function E(v,R){o.disableAll(),R.instancing&&o.enable(0),R.instancingColor&&o.enable(1),R.instancingMorph&&o.enable(2),R.matcap&&o.enable(3),R.envMap&&o.enable(4),R.normalMapObjectSpace&&o.enable(5),R.normalMapTangentSpace&&o.enable(6),R.clearcoat&&o.enable(7),R.iridescence&&o.enable(8),R.alphaTest&&o.enable(9),R.vertexColors&&o.enable(10),R.vertexAlphas&&o.enable(11),R.vertexUv1s&&o.enable(12),R.vertexUv2s&&o.enable(13),R.vertexUv3s&&o.enable(14),R.vertexTangents&&o.enable(15),R.anisotropy&&o.enable(16),R.alphaHash&&o.enable(17),R.batching&&o.enable(18),R.dispersion&&o.enable(19),R.batchingColor&&o.enable(20),R.gradientMap&&o.enable(21),R.packedNormalMap&&o.enable(22),R.vertexNormals&&o.enable(23),v.push(o.mask),o.disableAll(),R.fog&&o.enable(0),R.useFog&&o.enable(1),R.flatShading&&o.enable(2),R.logarithmicDepthBuffer&&o.enable(3),R.reversedDepthBuffer&&o.enable(4),R.skinning&&o.enable(5),R.morphTargets&&o.enable(6),R.morphNormals&&o.enable(7),R.morphColors&&o.enable(8),R.premultipliedAlpha&&o.enable(9),R.shadowMapEnabled&&o.enable(10),R.doubleSided&&o.enable(11),R.flipSided&&o.enable(12),R.useDepthPacking&&o.enable(13),R.dithering&&o.enable(14),R.transmission&&o.enable(15),R.sheen&&o.enable(16),R.opaque&&o.enable(17),R.pointsUvs&&o.enable(18),R.decodeVideoTexture&&o.enable(19),R.decodeVideoTextureEmissive&&o.enable(20),R.alphaToCoverage&&o.enable(21),R.numLightProbeGrids>0&&o.enable(22),R.hasPositionAttribute&&o.enable(23),v.push(o.mask)}function S(v){let R=d[v.type],F;if(R){let N=si[R];F=zf.clone(N.uniforms)}else F=v.uniforms;return F}function x(v,R){let F=h.get(R);return F!==void 0?++F.usedTimes:(F=new Vx(i,R,v,s),l.push(F),h.set(R,F)),F}function A(v){if(--v.usedTimes===0){let R=l.indexOf(v);l[R]=l[l.length-1],l.pop(),h.delete(v.cacheKey),v.destroy()}}function w(v){a.remove(v)}function I(){a.dispose()}return{getParameters:y,getProgramCacheKey:m,getUniforms:S,acquireProgram:x,releaseProgram:A,releaseShaderCache:w,programs:l,dispose:I}}function qx(){let i=new WeakMap;function e(o){return i.has(o)}function t(o){let a=i.get(o);return a===void 0&&(a={},i.set(o,a)),a}function n(o){i.delete(o)}function s(o,a,c){i.get(o)[a]=c}function r(){i=new WeakMap}return{has:e,get:t,remove:n,update:s,dispose:r}}function Yx(i,e){return i.groupOrder!==e.groupOrder?i.groupOrder-e.groupOrder:i.renderOrder!==e.renderOrder?i.renderOrder-e.renderOrder:i.material.id!==e.material.id?i.material.id-e.material.id:i.materialVariant!==e.materialVariant?i.materialVariant-e.materialVariant:i.z!==e.z?i.z-e.z:i.id-e.id}function od(i,e){return i.groupOrder!==e.groupOrder?i.groupOrder-e.groupOrder:i.renderOrder!==e.renderOrder?i.renderOrder-e.renderOrder:i.z!==e.z?e.z-i.z:i.id-e.id}function ad(){let i=[],e=0,t=[],n=[],s=[];function r(){e=0,t.length=0,n.length=0,s.length=0}function o(f){let d=0;return f.isInstancedMesh&&(d+=2),f.isSkinnedMesh&&(d+=1),d}function a(f,d,p,y,m,g){let E=i[e];return E===void 0?(E={id:f.id,object:f,geometry:d,material:p,materialVariant:o(f),groupOrder:y,renderOrder:f.renderOrder,z:m,group:g},i[e]=E):(E.id=f.id,E.object=f,E.geometry=d,E.material=p,E.materialVariant=o(f),E.groupOrder=y,E.renderOrder=f.renderOrder,E.z=m,E.group=g),e++,E}function c(f,d,p,y,m,g){let E=a(f,d,p,y,m,g);p.transmission>0?n.push(E):p.transparent===!0?s.push(E):t.push(E)}function l(f,d,p,y,m,g){let E=a(f,d,p,y,m,g);p.transmission>0?n.unshift(E):p.transparent===!0?s.unshift(E):t.unshift(E)}function h(f,d,p){t.length>1&&t.sort(f||Yx),n.length>1&&n.sort(d||od),s.length>1&&s.sort(d||od),p&&(t.reverse(),n.reverse(),s.reverse())}function u(){for(let f=e,d=i.length;f<d;f++){let p=i[f];if(p.id===null)break;p.id=null,p.object=null,p.geometry=null,p.material=null,p.group=null}}return{opaque:t,transmissive:n,transparent:s,init:r,push:c,unshift:l,finish:u,sort:h}}function Zx(){let i=new WeakMap;function e(n,s){let r=i.get(n),o;return r===void 0?(o=new ad,i.set(n,[o])):s>=r.length?(o=new ad,r.push(o)):o=r[s],o}function t(){i=new WeakMap}return{get:e,dispose:t}}function Kx(){let i={};return{get:function(e){if(i[e.id]!==void 0)return i[e.id];let t;switch(e.type){case"DirectionalLight":t={direction:new B,color:new Ge};break;case"SpotLight":t={position:new B,direction:new B,color:new Ge,distance:0,coneCos:0,penumbraCos:0,decay:0};break;case"PointLight":t={position:new B,color:new Ge,distance:0,decay:0};break;case"HemisphereLight":t={direction:new B,skyColor:new Ge,groundColor:new Ge};break;case"RectAreaLight":t={color:new Ge,position:new B,halfWidth:new B,halfHeight:new B};break}return i[e.id]=t,t}}}function jx(){let i={};return{get:function(e){if(i[e.id]!==void 0)return i[e.id];let t;switch(e.type){case"DirectionalLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new de};break;case"SpotLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new de};break;case"PointLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new de,shadowCameraNear:1,shadowCameraFar:1e3};break}return i[e.id]=t,t}}}var Jx=0;function $x(i,e){return(e.castShadow?2:0)-(i.castShadow?2:0)+(e.map?1:0)-(i.map?1:0)}function Qx(i){let e=new Kx,t=jx(),n={version:0,hash:{directionalLength:-1,pointLength:-1,spotLength:-1,rectAreaLength:-1,hemiLength:-1,numDirectionalShadows:-1,numPointShadows:-1,numSpotShadows:-1,numSpotMaps:-1,numLightProbes:-1},ambient:[0,0,0],probe:[],directional:[],directionalShadow:[],directionalShadowMap:[],directionalShadowMatrix:[],spot:[],spotLightMap:[],spotShadow:[],spotShadowMap:[],spotLightMatrix:[],rectArea:[],rectAreaLTC1:null,rectAreaLTC2:null,point:[],pointShadow:[],pointShadowMap:[],pointShadowMatrix:[],hemi:[],numSpotLightShadowsWithMaps:0,numLightProbes:0};for(let l=0;l<9;l++)n.probe.push(new B);let s=new B,r=new Je,o=new Je;function a(l){let h=0,u=0,f=0;for(let R=0;R<9;R++)n.probe[R].set(0,0,0);let d=0,p=0,y=0,m=0,g=0,E=0,S=0,x=0,A=0,w=0,I=0;l.sort($x);for(let R=0,F=l.length;R<F;R++){let N=l[R],H=N.color,oe=N.intensity,ie=N.distance,X=null;if(N.shadow&&N.shadow.map&&(N.shadow.map.texture.format===Ji?X=N.shadow.map.texture:X=N.shadow.map.depthTexture||N.shadow.map.texture),N.isAmbientLight)h+=H.r*oe,u+=H.g*oe,f+=H.b*oe;else if(N.isLightProbe){for(let ee=0;ee<9;ee++)n.probe[ee].addScaledVector(N.sh.coefficients[ee],oe);I++}else if(N.isDirectionalLight){let ee=e.get(N);if(ee.color.copy(N.color).multiplyScalar(N.intensity),N.castShadow){let Y=N.shadow,G=t.get(N);G.shadowIntensity=Y.intensity,G.shadowBias=Y.bias,G.shadowNormalBias=Y.normalBias,G.shadowRadius=Y.radius,G.shadowMapSize=Y.mapSize,n.directionalShadow[d]=G,n.directionalShadowMap[d]=X,n.directionalShadowMatrix[d]=N.shadow.matrix,E++}n.directional[d]=ee,d++}else if(N.isSpotLight){let ee=e.get(N);ee.position.setFromMatrixPosition(N.matrixWorld),ee.color.copy(H).multiplyScalar(oe),ee.distance=ie,ee.coneCos=Math.cos(N.angle),ee.penumbraCos=Math.cos(N.angle*(1-N.penumbra)),ee.decay=N.decay,n.spot[y]=ee;let Y=N.shadow;if(N.map&&(n.spotLightMap[A]=N.map,A++,Y.updateMatrices(N),N.castShadow&&w++),n.spotLightMatrix[y]=Y.matrix,N.castShadow){let G=t.get(N);G.shadowIntensity=Y.intensity,G.shadowBias=Y.bias,G.shadowNormalBias=Y.normalBias,G.shadowRadius=Y.radius,G.shadowMapSize=Y.mapSize,n.spotShadow[y]=G,n.spotShadowMap[y]=X,x++}y++}else if(N.isRectAreaLight){let ee=e.get(N);ee.color.copy(H).multiplyScalar(oe),ee.halfWidth.set(N.width*.5,0,0),ee.halfHeight.set(0,N.height*.5,0),n.rectArea[m]=ee,m++}else if(N.isPointLight){let ee=e.get(N);if(ee.color.copy(N.color).multiplyScalar(N.intensity),ee.distance=N.distance,ee.decay=N.decay,N.castShadow){let Y=N.shadow,G=t.get(N);G.shadowIntensity=Y.intensity,G.shadowBias=Y.bias,G.shadowNormalBias=Y.normalBias,G.shadowRadius=Y.radius,G.shadowMapSize=Y.mapSize,G.shadowCameraNear=Y.camera.near,G.shadowCameraFar=Y.camera.far,n.pointShadow[p]=G,n.pointShadowMap[p]=X,n.pointShadowMatrix[p]=N.shadow.matrix,S++}n.point[p]=ee,p++}else if(N.isHemisphereLight){let ee=e.get(N);ee.skyColor.copy(N.color).multiplyScalar(oe),ee.groundColor.copy(N.groundColor).multiplyScalar(oe),n.hemi[g]=ee,g++}}m>0&&(i.has("OES_texture_float_linear")===!0?(n.rectAreaLTC1=Le.LTC_FLOAT_1,n.rectAreaLTC2=Le.LTC_FLOAT_2):(n.rectAreaLTC1=Le.LTC_HALF_1,n.rectAreaLTC2=Le.LTC_HALF_2)),n.ambient[0]=h,n.ambient[1]=u,n.ambient[2]=f;let v=n.hash;(v.directionalLength!==d||v.pointLength!==p||v.spotLength!==y||v.rectAreaLength!==m||v.hemiLength!==g||v.numDirectionalShadows!==E||v.numPointShadows!==S||v.numSpotShadows!==x||v.numSpotMaps!==A||v.numLightProbes!==I)&&(n.directional.length=d,n.spot.length=y,n.rectArea.length=m,n.point.length=p,n.hemi.length=g,n.directionalShadow.length=E,n.directionalShadowMap.length=E,n.pointShadow.length=S,n.pointShadowMap.length=S,n.spotShadow.length=x,n.spotShadowMap.length=x,n.directionalShadowMatrix.length=E,n.pointShadowMatrix.length=S,n.spotLightMatrix.length=x+A-w,n.spotLightMap.length=A,n.numSpotLightShadowsWithMaps=w,n.numLightProbes=I,v.directionalLength=d,v.pointLength=p,v.spotLength=y,v.rectAreaLength=m,v.hemiLength=g,v.numDirectionalShadows=E,v.numPointShadows=S,v.numSpotShadows=x,v.numSpotMaps=A,v.numLightProbes=I,n.version=Jx++)}function c(l,h){let u=0,f=0,d=0,p=0,y=0,m=h.matrixWorldInverse;for(let g=0,E=l.length;g<E;g++){let S=l[g];if(S.isDirectionalLight){let x=n.directional[u];x.direction.setFromMatrixPosition(S.matrixWorld),s.setFromMatrixPosition(S.target.matrixWorld),x.direction.sub(s),x.direction.transformDirection(m),u++}else if(S.isSpotLight){let x=n.spot[d];x.position.setFromMatrixPosition(S.matrixWorld),x.position.applyMatrix4(m),x.direction.setFromMatrixPosition(S.matrixWorld),s.setFromMatrixPosition(S.target.matrixWorld),x.direction.sub(s),x.direction.transformDirection(m),d++}else if(S.isRectAreaLight){let x=n.rectArea[p];x.position.setFromMatrixPosition(S.matrixWorld),x.position.applyMatrix4(m),o.identity(),r.copy(S.matrixWorld),r.premultiply(m),o.extractRotation(r),x.halfWidth.set(S.width*.5,0,0),x.halfHeight.set(0,S.height*.5,0),x.halfWidth.applyMatrix4(o),x.halfHeight.applyMatrix4(o),p++}else if(S.isPointLight){let x=n.point[f];x.position.setFromMatrixPosition(S.matrixWorld),x.position.applyMatrix4(m),f++}else if(S.isHemisphereLight){let x=n.hemi[y];x.direction.setFromMatrixPosition(S.matrixWorld),x.direction.transformDirection(m),y++}}}return{setup:a,setupView:c,state:n}}function cd(i){let e=new Qx(i),t=[],n=[],s=[];function r(f){u.camera=f,t.length=0,n.length=0,s.length=0}function o(f){t.push(f)}function a(f){n.push(f)}function c(f){s.push(f)}function l(){e.setup(t)}function h(f){e.setupView(t,f)}let u={lightsArray:t,shadowsArray:n,lightProbeGridArray:s,camera:null,lights:e,transmissionRenderTarget:{},textureUnits:0};return{init:r,state:u,setupLights:l,setupLightsView:h,pushLight:o,pushShadow:a,pushLightProbeGrid:c}}function ey(i){let e=new WeakMap;function t(s,r=0){let o=e.get(s),a;return o===void 0?(a=new cd(i),e.set(s,[a])):r>=o.length?(a=new cd(i),o.push(a)):a=o[r],a}function n(){e=new WeakMap}return{get:t,dispose:n}}var ty=`void main() {
	gl_Position = vec4( position, 1.0 );
}`,ny=`uniform sampler2D shadow_pass;
uniform vec2 resolution;
uniform float radius;
void main() {
	const float samples = float( VSM_SAMPLES );
	float mean = 0.0;
	float squared_mean = 0.0;
	float uvStride = samples <= 1.0 ? 0.0 : 2.0 / ( samples - 1.0 );
	float uvStart = samples <= 1.0 ? 0.0 : - 1.0;
	for ( float i = 0.0; i < samples; i ++ ) {
		float uvOffset = uvStart + i * uvStride;
		#ifdef HORIZONTAL_PASS
			vec2 distribution = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( uvOffset, 0.0 ) * radius ) / resolution ).rg;
			mean += distribution.x;
			squared_mean += distribution.y * distribution.y + distribution.x * distribution.x;
		#else
			float depth = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( 0.0, uvOffset ) * radius ) / resolution ).r;
			mean += depth;
			squared_mean += depth * depth;
		#endif
	}
	mean = mean / samples;
	squared_mean = squared_mean / samples;
	float std_dev = sqrt( max( 0.0, squared_mean - mean * mean ) );
	gl_FragColor = vec4( mean, std_dev, 0.0, 1.0 );
}`,iy=[new B(1,0,0),new B(-1,0,0),new B(0,1,0),new B(0,-1,0),new B(0,0,1),new B(0,0,-1)],sy=[new B(0,-1,0),new B(0,-1,0),new B(0,0,1),new B(0,0,-1),new B(0,-1,0),new B(0,-1,0)],ld=new Je,Eo=new B,dh=new B;function ry(i,e,t){let n=new er,s=new de,r=new de,o=new ft,a=new Ca,c=new Pa,l={},h=t.maxTextureSize,u={[On]:Ft,[Ft]:On,[Bt]:Bt},f=new Jt({defines:{VSM_SAMPLES:8},uniforms:{shadow_pass:{value:null},resolution:{value:new de},radius:{value:4}},vertexShader:ty,fragmentShader:ny}),d=f.clone();d.defines.HORIZONTAL_PASS=1;let p=new wt;p.setAttribute("position",new yt(new Float32Array([-1,-1,.5,3,-1,.5,-1,3,.5]),3));let y=new ht(p,f),m=this;this.enabled=!1,this.autoUpdate=!0,this.needsUpdate=!1,this.type=mo;let g=this.type;this.render=function(w,I,v){if(m.enabled===!1||m.autoUpdate===!1&&m.needsUpdate===!1||w.length===0)return;this.type===Qu&&(ke("WebGLShadowMap: PCFSoftShadowMap has been deprecated. Using PCFShadowMap instead."),this.type=mo);let R=i.getRenderTarget(),F=i.getActiveCubeFace(),N=i.getActiveMipmapLevel(),H=i.state;H.setBlending(xn),H.buffers.depth.getReversed()===!0?H.buffers.color.setClear(0,0,0,0):H.buffers.color.setClear(1,1,1,1),H.buffers.depth.setTest(!0),H.setScissorTest(!1);let oe=g!==this.type;oe&&I.traverse(function(ie){ie.material&&(Array.isArray(ie.material)?ie.material.forEach(X=>X.needsUpdate=!0):ie.material.needsUpdate=!0)});for(let ie=0,X=w.length;ie<X;ie++){let ee=w[ie],Y=ee.shadow;if(Y===void 0){ke("WebGLShadowMap:",ee,"has no shadow.");continue}if(Y.autoUpdate===!1&&Y.needsUpdate===!1)continue;s.copy(Y.mapSize);let G=Y.getFrameExtents();s.multiply(G),r.copy(Y.mapSize),(s.x>h||s.y>h)&&(s.x>h&&(r.x=Math.floor(h/G.x),s.x=r.x*G.x,Y.mapSize.x=r.x),s.y>h&&(r.y=Math.floor(h/G.y),s.y=r.y*G.y,Y.mapSize.y=r.y));let ge=i.state.buffers.depth.getReversed();if(Y.camera._reversedDepth=ge,Y.map===null||oe===!0){if(Y.map!==null&&(Y.map.depthTexture!==null&&(Y.map.depthTexture.dispose(),Y.map.depthTexture=null),Y.map.dispose()),this.type===ar){if(ee.isPointLight){ke("WebGLShadowMap: VSM shadow maps are not supported for PointLights. Use PCF or BasicShadowMap instead.");continue}Y.map=new jt(s.x,s.y,{format:Ji,type:ni,minFilter:bt,magFilter:bt,generateMipmaps:!1}),Y.map.texture.name=ee.name+".shadowMap",Y.map.depthTexture=new xi(s.x,s.y,nn),Y.map.depthTexture.name=ee.name+".shadowMapDepth",Y.map.depthTexture.format=$n,Y.map.depthTexture.compareFunction=null,Y.map.depthTexture.minFilter=St,Y.map.depthTexture.magFilter=St}else ee.isPointLight?(Y.map=new Pc(s.x),Y.map.depthTexture=new ba(s.x,Wn)):(Y.map=new jt(s.x,s.y),Y.map.depthTexture=new xi(s.x,s.y,Wn)),Y.map.depthTexture.name=ee.name+".shadowMap",Y.map.depthTexture.format=$n,this.type===mo?(Y.map.depthTexture.compareFunction=ge?wc:Ac,Y.map.depthTexture.minFilter=bt,Y.map.depthTexture.magFilter=bt):(Y.map.depthTexture.compareFunction=null,Y.map.depthTexture.minFilter=St,Y.map.depthTexture.magFilter=St);Y.camera.updateProjectionMatrix()}let ye=Y.map.isWebGLCubeRenderTarget?6:1;for(let Me=0;Me<ye;Me++){if(Y.map.isWebGLCubeRenderTarget)i.setRenderTarget(Y.map,Me),i.clear();else{Me===0&&(i.setRenderTarget(Y.map),i.clear());let Re=Y.getViewport(Me);o.set(r.x*Re.x,r.y*Re.y,r.x*Re.z,r.y*Re.w),H.viewport(o)}if(ee.isPointLight){let Re=Y.camera,ze=Y.matrix,qe=ee.distance||Re.far;qe!==Re.far&&(Re.far=qe,Re.updateProjectionMatrix()),Eo.setFromMatrixPosition(ee.matrixWorld),Re.position.copy(Eo),dh.copy(Re.position),dh.add(iy[Me]),Re.up.copy(sy[Me]),Re.lookAt(dh),Re.updateMatrixWorld(),ze.makeTranslation(-Eo.x,-Eo.y,-Eo.z),ld.multiplyMatrices(Re.projectionMatrix,Re.matrixWorldInverse),Y._frustum.setFromProjectionMatrix(ld,Re.coordinateSystem,Re.reversedDepth)}else Y.updateMatrices(ee);n=Y.getFrustum(),x(I,v,Y.camera,ee,this.type)}Y.isPointLightShadow!==!0&&this.type===ar&&E(Y,v),Y.needsUpdate=!1}g=this.type,m.needsUpdate=!1,i.setRenderTarget(R,F,N)};function E(w,I){let v=e.update(y);f.defines.VSM_SAMPLES!==w.blurSamples&&(f.defines.VSM_SAMPLES=w.blurSamples,d.defines.VSM_SAMPLES=w.blurSamples,f.needsUpdate=!0,d.needsUpdate=!0),w.mapPass===null&&(w.mapPass=new jt(s.x,s.y,{format:Ji,type:ni})),f.uniforms.shadow_pass.value=w.map.depthTexture,f.uniforms.resolution.value=w.mapSize,f.uniforms.radius.value=w.radius,i.setRenderTarget(w.mapPass),i.clear(),i.renderBufferDirect(I,null,v,f,y,null),d.uniforms.shadow_pass.value=w.mapPass.texture,d.uniforms.resolution.value=w.mapSize,d.uniforms.radius.value=w.radius,i.setRenderTarget(w.map),i.clear(),i.renderBufferDirect(I,null,v,d,y,null)}function S(w,I,v,R){let F=null,N=v.isPointLight===!0?w.customDistanceMaterial:w.customDepthMaterial;if(N!==void 0)F=N;else if(F=v.isPointLight===!0?c:a,i.localClippingEnabled&&I.clipShadows===!0&&Array.isArray(I.clippingPlanes)&&I.clippingPlanes.length!==0||I.displacementMap&&I.displacementScale!==0||I.alphaMap&&I.alphaTest>0||I.map&&I.alphaTest>0||I.alphaToCoverage===!0){let H=F.uuid,oe=I.uuid,ie=l[H];ie===void 0&&(ie={},l[H]=ie);let X=ie[oe];X===void 0&&(X=F.clone(),ie[oe]=X,I.addEventListener("dispose",A)),F=X}if(F.visible=I.visible,F.wireframe=I.wireframe,R===ar?F.side=I.shadowSide!==null?I.shadowSide:I.side:F.side=I.shadowSide!==null?I.shadowSide:u[I.side],F.alphaMap=I.alphaMap,F.alphaTest=I.alphaToCoverage===!0?.5:I.alphaTest,F.map=I.map,F.clipShadows=I.clipShadows,F.clippingPlanes=I.clippingPlanes,F.clipIntersection=I.clipIntersection,F.displacementMap=I.displacementMap,F.displacementScale=I.displacementScale,F.displacementBias=I.displacementBias,F.wireframeLinewidth=I.wireframeLinewidth,F.linewidth=I.linewidth,v.isPointLight===!0&&F.isMeshDistanceMaterial===!0){let H=i.properties.get(F);H.light=v}return F}function x(w,I,v,R,F){if(w.visible===!1)return;if(w.layers.test(I.layers)&&(w.isMesh||w.isLine||w.isPoints)&&(w.castShadow||w.receiveShadow&&F===ar)&&(!w.frustumCulled||n.intersectsObject(w))){w.modelViewMatrix.multiplyMatrices(v.matrixWorldInverse,w.matrixWorld);let oe=e.update(w),ie=w.material;if(Array.isArray(ie)){let X=oe.groups;for(let ee=0,Y=X.length;ee<Y;ee++){let G=X[ee],ge=ie[G.materialIndex];if(ge&&ge.visible){let ye=S(w,ge,R,F);w.onBeforeShadow(i,w,I,v,oe,ye,G),i.renderBufferDirect(v,null,oe,ye,w,G),w.onAfterShadow(i,w,I,v,oe,ye,G)}}}else if(ie.visible){let X=S(w,ie,R,F);w.onBeforeShadow(i,w,I,v,oe,X,null),i.renderBufferDirect(v,null,oe,X,w,null),w.onAfterShadow(i,w,I,v,oe,X,null)}}let H=w.children;for(let oe=0,ie=H.length;oe<ie;oe++)x(H[oe],I,v,R,F)}function A(w){w.target.removeEventListener("dispose",A);for(let v in l){let R=l[v],F=w.target.uuid;F in R&&(R[F].dispose(),delete R[F])}}}function oy(i,e){function t(){let Z=!1,Pe=new ft,be=null,Ie=new ft(0,0,0,0);return{setMask:function(Ce){be!==Ce&&!Z&&(i.colorMask(Ce,Ce,Ce,Ce),be=Ce)},setLocked:function(Ce){Z=Ce},setClear:function(Ce,Te,He,Be,dt){dt===!0&&(Ce*=Be,Te*=Be,He*=Be),Pe.set(Ce,Te,He,Be),Ie.equals(Pe)===!1&&(i.clearColor(Ce,Te,He,Be),Ie.copy(Pe))},reset:function(){Z=!1,be=null,Ie.set(-1,0,0,0)}}}function n(){let Z=!1,Pe=!1,be=null,Ie=null,Ce=null;return{setReversed:function(Te){if(Pe!==Te){let He=e.get("EXT_clip_control");Te?He.clipControlEXT(He.LOWER_LEFT_EXT,He.ZERO_TO_ONE_EXT):He.clipControlEXT(He.LOWER_LEFT_EXT,He.NEGATIVE_ONE_TO_ONE_EXT),Pe=Te;let Be=Ce;Ce=null,this.setClear(Be)}},getReversed:function(){return Pe},setTest:function(Te){Te?U(i.DEPTH_TEST):L(i.DEPTH_TEST)},setMask:function(Te){be!==Te&&!Z&&(i.depthMask(Te),be=Te)},setFunc:function(Te){if(Pe&&(Te=Nf[Te]),Ie!==Te){switch(Te){case ca:i.depthFunc(i.NEVER);break;case la:i.depthFunc(i.ALWAYS);break;case ha:i.depthFunc(i.LESS);break;case us:i.depthFunc(i.LEQUAL);break;case ua:i.depthFunc(i.EQUAL);break;case fa:i.depthFunc(i.GEQUAL);break;case da:i.depthFunc(i.GREATER);break;case pa:i.depthFunc(i.NOTEQUAL);break;default:i.depthFunc(i.LEQUAL)}Ie=Te}},setLocked:function(Te){Z=Te},setClear:function(Te){Ce!==Te&&(Ce=Te,Pe&&(Te=1-Te),i.clearDepth(Te))},reset:function(){Z=!1,be=null,Ie=null,Ce=null,Pe=!1}}}function s(){let Z=!1,Pe=null,be=null,Ie=null,Ce=null,Te=null,He=null,Be=null,dt=null;return{setTest:function(ot){Z||(ot?U(i.STENCIL_TEST):L(i.STENCIL_TEST))},setMask:function(ot){Pe!==ot&&!Z&&(i.stencilMask(ot),Pe=ot)},setFunc:function(ot,sn,ct){(be!==ot||Ie!==sn||Ce!==ct)&&(i.stencilFunc(ot,sn,ct),be=ot,Ie=sn,Ce=ct)},setOp:function(ot,sn,ct){(Te!==ot||He!==sn||Be!==ct)&&(i.stencilOp(ot,sn,ct),Te=ot,He=sn,Be=ct)},setLocked:function(ot){Z=ot},setClear:function(ot){dt!==ot&&(i.clearStencil(ot),dt=ot)},reset:function(){Z=!1,Pe=null,be=null,Ie=null,Ce=null,Te=null,He=null,Be=null,dt=null}}}let r=new t,o=new n,a=new s,c=new WeakMap,l=new WeakMap,h={},u={},f={},d=new WeakMap,p=[],y=null,m=!1,g=null,E=null,S=null,x=null,A=null,w=null,I=null,v=new Ge(0,0,0),R=0,F=!1,N=null,H=null,oe=null,ie=null,X=null,ee=i.getParameter(i.MAX_COMBINED_TEXTURE_IMAGE_UNITS),Y=!1,G=0,ge=i.getParameter(i.VERSION);ge.indexOf("WebGL")!==-1?(G=parseFloat(/^WebGL (\d)/.exec(ge)[1]),Y=G>=1):ge.indexOf("OpenGL ES")!==-1&&(G=parseFloat(/^OpenGL ES (\d)/.exec(ge)[1]),Y=G>=2);let ye=null,Me={},Re=i.getParameter(i.SCISSOR_BOX),ze=i.getParameter(i.VIEWPORT),qe=new ft().fromArray(Re),We=new ft().fromArray(ze);function pe(Z,Pe,be,Ie){let Ce=new Uint8Array(4),Te=i.createTexture();i.bindTexture(Z,Te),i.texParameteri(Z,i.TEXTURE_MIN_FILTER,i.NEAREST),i.texParameteri(Z,i.TEXTURE_MAG_FILTER,i.NEAREST);for(let He=0;He<be;He++)Z===i.TEXTURE_3D||Z===i.TEXTURE_2D_ARRAY?i.texImage3D(Pe,0,i.RGBA,1,1,Ie,0,i.RGBA,i.UNSIGNED_BYTE,Ce):i.texImage2D(Pe+He,0,i.RGBA,1,1,0,i.RGBA,i.UNSIGNED_BYTE,Ce);return Te}let q={};q[i.TEXTURE_2D]=pe(i.TEXTURE_2D,i.TEXTURE_2D,1),q[i.TEXTURE_CUBE_MAP]=pe(i.TEXTURE_CUBE_MAP,i.TEXTURE_CUBE_MAP_POSITIVE_X,6),q[i.TEXTURE_2D_ARRAY]=pe(i.TEXTURE_2D_ARRAY,i.TEXTURE_2D_ARRAY,1,1),q[i.TEXTURE_3D]=pe(i.TEXTURE_3D,i.TEXTURE_3D,1,1),r.setClear(0,0,0,1),o.setClear(1),a.setClear(0),U(i.DEPTH_TEST),o.setFunc(us),j(!1),$(Ll),U(i.CULL_FACE),V(xn);function U(Z){h[Z]!==!0&&(i.enable(Z),h[Z]=!0)}function L(Z){h[Z]!==!1&&(i.disable(Z),h[Z]=!1)}function P(Z,Pe){return f[Z]!==Pe?(i.bindFramebuffer(Z,Pe),f[Z]=Pe,Z===i.DRAW_FRAMEBUFFER&&(f[i.FRAMEBUFFER]=Pe),Z===i.FRAMEBUFFER&&(f[i.DRAW_FRAMEBUFFER]=Pe),!0):!1}function z(Z,Pe){let be=p,Ie=!1;if(Z){be=d.get(Pe),be===void 0&&(be=[],d.set(Pe,be));let Ce=Z.textures;if(be.length!==Ce.length||be[0]!==i.COLOR_ATTACHMENT0){for(let Te=0,He=Ce.length;Te<He;Te++)be[Te]=i.COLOR_ATTACHMENT0+Te;be.length=Ce.length,Ie=!0}}else be[0]!==i.BACK&&(be[0]=i.BACK,Ie=!0);Ie&&i.drawBuffers(be)}function he(Z){return y!==Z?(i.useProgram(Z),y=Z,!0):!1}let me={[ki]:i.FUNC_ADD,[tf]:i.FUNC_SUBTRACT,[nf]:i.FUNC_REVERSE_SUBTRACT};me[sf]=i.MIN,me[rf]=i.MAX;let C={[of]:i.ZERO,[af]:i.ONE,[cf]:i.SRC_COLOR,[oa]:i.SRC_ALPHA,[pf]:i.SRC_ALPHA_SATURATE,[ff]:i.DST_COLOR,[hf]:i.DST_ALPHA,[lf]:i.ONE_MINUS_SRC_COLOR,[aa]:i.ONE_MINUS_SRC_ALPHA,[df]:i.ONE_MINUS_DST_COLOR,[uf]:i.ONE_MINUS_DST_ALPHA,[mf]:i.CONSTANT_COLOR,[gf]:i.ONE_MINUS_CONSTANT_COLOR,[_f]:i.CONSTANT_ALPHA,[xf]:i.ONE_MINUS_CONSTANT_ALPHA};function V(Z,Pe,be,Ie,Ce,Te,He,Be,dt,ot){if(Z===xn){m===!0&&(L(i.BLEND),m=!1);return}if(m===!1&&(U(i.BLEND),m=!0),Z!==ef){if(Z!==g||ot!==F){if((E!==ki||A!==ki)&&(i.blendEquation(i.FUNC_ADD),E=ki,A=ki),ot)switch(Z){case hs:i.blendFuncSeparate(i.ONE,i.ONE_MINUS_SRC_ALPHA,i.ONE,i.ONE_MINUS_SRC_ALPHA);break;case Dl:i.blendFunc(i.ONE,i.ONE);break;case Nl:i.blendFuncSeparate(i.ZERO,i.ONE_MINUS_SRC_COLOR,i.ZERO,i.ONE);break;case Ul:i.blendFuncSeparate(i.DST_COLOR,i.ONE_MINUS_SRC_ALPHA,i.ZERO,i.ONE);break;default:Ke("WebGLState: Invalid blending: ",Z);break}else switch(Z){case hs:i.blendFuncSeparate(i.SRC_ALPHA,i.ONE_MINUS_SRC_ALPHA,i.ONE,i.ONE_MINUS_SRC_ALPHA);break;case Dl:i.blendFuncSeparate(i.SRC_ALPHA,i.ONE,i.ONE,i.ONE);break;case Nl:Ke("WebGLState: SubtractiveBlending requires material.premultipliedAlpha = true");break;case Ul:Ke("WebGLState: MultiplyBlending requires material.premultipliedAlpha = true");break;default:Ke("WebGLState: Invalid blending: ",Z);break}S=null,x=null,w=null,I=null,v.set(0,0,0),R=0,g=Z,F=ot}return}Ce=Ce||Pe,Te=Te||be,He=He||Ie,(Pe!==E||Ce!==A)&&(i.blendEquationSeparate(me[Pe],me[Ce]),E=Pe,A=Ce),(be!==S||Ie!==x||Te!==w||He!==I)&&(i.blendFuncSeparate(C[be],C[Ie],C[Te],C[He]),S=be,x=Ie,w=Te,I=He),(Be.equals(v)===!1||dt!==R)&&(i.blendColor(Be.r,Be.g,Be.b,dt),v.copy(Be),R=dt),g=Z,F=!1}function D(Z,Pe){Z.side===Bt?L(i.CULL_FACE):U(i.CULL_FACE);let be=Z.side===Ft;Pe&&(be=!be),j(be),Z.blending===hs&&Z.transparent===!1?V(xn):V(Z.blending,Z.blendEquation,Z.blendSrc,Z.blendDst,Z.blendEquationAlpha,Z.blendSrcAlpha,Z.blendDstAlpha,Z.blendColor,Z.blendAlpha,Z.premultipliedAlpha),o.setFunc(Z.depthFunc),o.setTest(Z.depthTest),o.setMask(Z.depthWrite),r.setMask(Z.colorWrite);let Ie=Z.stencilWrite;a.setTest(Ie),Ie&&(a.setMask(Z.stencilWriteMask),a.setFunc(Z.stencilFunc,Z.stencilRef,Z.stencilFuncMask),a.setOp(Z.stencilFail,Z.stencilZFail,Z.stencilZPass)),te(Z.polygonOffset,Z.polygonOffsetFactor,Z.polygonOffsetUnits),Z.alphaToCoverage===!0?U(i.SAMPLE_ALPHA_TO_COVERAGE):L(i.SAMPLE_ALPHA_TO_COVERAGE)}function j(Z){N!==Z&&(Z?i.frontFace(i.CW):i.frontFace(i.CCW),N=Z)}function $(Z){Z!==Ju?(U(i.CULL_FACE),Z!==H&&(Z===Ll?i.cullFace(i.BACK):Z===$u?i.cullFace(i.FRONT):i.cullFace(i.FRONT_AND_BACK))):L(i.CULL_FACE),H=Z}function le(Z){Z!==oe&&(Y&&i.lineWidth(Z),oe=Z)}function te(Z,Pe,be){Z?(U(i.POLYGON_OFFSET_FILL),(ie!==Pe||X!==be)&&(ie=Pe,X=be,o.getReversed()&&(Pe=-Pe),i.polygonOffset(Pe,be))):L(i.POLYGON_OFFSET_FILL)}function ne(Z){Z?U(i.SCISSOR_TEST):L(i.SCISSOR_TEST)}function k(Z){Z===void 0&&(Z=i.TEXTURE0+ee-1),ye!==Z&&(i.activeTexture(Z),ye=Z)}function b(Z,Pe,be){be===void 0&&(ye===null?be=i.TEXTURE0+ee-1:be=ye);let Ie=Me[be];Ie===void 0&&(Ie={type:void 0,texture:void 0},Me[be]=Ie),(Ie.type!==Z||Ie.texture!==Pe)&&(ye!==be&&(i.activeTexture(be),ye=be),i.bindTexture(Z,Pe||q[Z]),Ie.type=Z,Ie.texture=Pe)}function we(){let Z=Me[ye];Z!==void 0&&Z.type!==void 0&&(i.bindTexture(Z.type,null),Z.type=void 0,Z.texture=void 0)}function Ae(){try{i.compressedTexImage2D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function T(){try{i.compressedTexImage3D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function _(){try{i.texSubImage2D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function O(){try{i.texSubImage3D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function W(){try{i.compressedTexSubImage2D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function Q(){try{i.compressedTexSubImage3D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function _e(){try{i.texStorage2D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function Se(){try{i.texStorage3D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function ae(){try{i.texImage2D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function fe(){try{i.texImage3D(...arguments)}catch(Z){Ke("WebGLState:",Z)}}function J(Z){return u[Z]!==void 0?u[Z]:i.getParameter(Z)}function xe(Z,Pe){u[Z]!==Pe&&(i.pixelStorei(Z,Pe),u[Z]=Pe)}function ue(Z){qe.equals(Z)===!1&&(i.scissor(Z.x,Z.y,Z.z,Z.w),qe.copy(Z))}function ve(Z){We.equals(Z)===!1&&(i.viewport(Z.x,Z.y,Z.z,Z.w),We.copy(Z))}function Ee(Z,Pe){let be=l.get(Pe);be===void 0&&(be=new WeakMap,l.set(Pe,be));let Ie=be.get(Z);Ie===void 0&&(Ie=i.getUniformBlockIndex(Pe,Z.name),be.set(Z,Ie))}function De(Z,Pe){let Ie=l.get(Pe).get(Z);c.get(Pe)!==Ie&&(i.uniformBlockBinding(Pe,Ie,Z.__bindingPointIndex),c.set(Pe,Ie))}function je(){i.disable(i.BLEND),i.disable(i.CULL_FACE),i.disable(i.DEPTH_TEST),i.disable(i.POLYGON_OFFSET_FILL),i.disable(i.SCISSOR_TEST),i.disable(i.STENCIL_TEST),i.disable(i.SAMPLE_ALPHA_TO_COVERAGE),i.blendEquation(i.FUNC_ADD),i.blendFunc(i.ONE,i.ZERO),i.blendFuncSeparate(i.ONE,i.ZERO,i.ONE,i.ZERO),i.blendColor(0,0,0,0),i.colorMask(!0,!0,!0,!0),i.clearColor(0,0,0,0),i.depthMask(!0),i.depthFunc(i.LESS),o.setReversed(!1),i.clearDepth(1),i.stencilMask(4294967295),i.stencilFunc(i.ALWAYS,0,4294967295),i.stencilOp(i.KEEP,i.KEEP,i.KEEP),i.clearStencil(0),i.cullFace(i.BACK),i.frontFace(i.CCW),i.polygonOffset(0,0),i.activeTexture(i.TEXTURE0),i.bindFramebuffer(i.FRAMEBUFFER,null),i.bindFramebuffer(i.DRAW_FRAMEBUFFER,null),i.bindFramebuffer(i.READ_FRAMEBUFFER,null),i.useProgram(null),i.lineWidth(1),i.scissor(0,0,i.canvas.width,i.canvas.height),i.viewport(0,0,i.canvas.width,i.canvas.height),i.pixelStorei(i.PACK_ALIGNMENT,4),i.pixelStorei(i.UNPACK_ALIGNMENT,4),i.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,!1),i.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,!1),i.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,i.BROWSER_DEFAULT_WEBGL),i.pixelStorei(i.PACK_ROW_LENGTH,0),i.pixelStorei(i.PACK_SKIP_PIXELS,0),i.pixelStorei(i.PACK_SKIP_ROWS,0),i.pixelStorei(i.UNPACK_ROW_LENGTH,0),i.pixelStorei(i.UNPACK_IMAGE_HEIGHT,0),i.pixelStorei(i.UNPACK_SKIP_PIXELS,0),i.pixelStorei(i.UNPACK_SKIP_ROWS,0),i.pixelStorei(i.UNPACK_SKIP_IMAGES,0),h={},u={},ye=null,Me={},f={},d=new WeakMap,p=[],y=null,m=!1,g=null,E=null,S=null,x=null,A=null,w=null,I=null,v=new Ge(0,0,0),R=0,F=!1,N=null,H=null,oe=null,ie=null,X=null,qe.set(0,0,i.canvas.width,i.canvas.height),We.set(0,0,i.canvas.width,i.canvas.height),r.reset(),o.reset(),a.reset()}return{buffers:{color:r,depth:o,stencil:a},enable:U,disable:L,bindFramebuffer:P,drawBuffers:z,useProgram:he,setBlending:V,setMaterial:D,setFlipSided:j,setCullFace:$,setLineWidth:le,setPolygonOffset:te,setScissorTest:ne,activeTexture:k,bindTexture:b,unbindTexture:we,compressedTexImage2D:Ae,compressedTexImage3D:T,texImage2D:ae,texImage3D:fe,pixelStorei:xe,getParameter:J,updateUBOMapping:Ee,uniformBlockBinding:De,texStorage2D:_e,texStorage3D:Se,texSubImage2D:_,texSubImage3D:O,compressedTexSubImage2D:W,compressedTexSubImage3D:Q,scissor:ue,viewport:ve,reset:je}}function ay(i,e,t,n,s,r,o){let a=e.has("WEBGL_multisampled_render_to_texture")?e.get("WEBGL_multisampled_render_to_texture"):null,c=typeof navigator>"u"?!1:/OculusBrowser/g.test(navigator.userAgent),l=new de,h=new WeakMap,u=new Set,f,d=new WeakMap,p=!1;try{p=typeof OffscreenCanvas<"u"&&new OffscreenCanvas(1,1).getContext("2d")!==null}catch{}function y(T,_){return p?new OffscreenCanvas(T,_):Ks("canvas")}function m(T,_,O){let W=1,Q=Ae(T);if((Q.width>O||Q.height>O)&&(W=O/Math.max(Q.width,Q.height)),W<1)if(typeof HTMLImageElement<"u"&&T instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&T instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&T instanceof ImageBitmap||typeof VideoFrame<"u"&&T instanceof VideoFrame){let _e=Math.floor(W*Q.width),Se=Math.floor(W*Q.height);f===void 0&&(f=y(_e,Se));let ae=_?y(_e,Se):f;return ae.width=_e,ae.height=Se,ae.getContext("2d").drawImage(T,0,0,_e,Se),ke("WebGLRenderer: Texture has been resized from ("+Q.width+"x"+Q.height+") to ("+_e+"x"+Se+")."),ae}else return"data"in T&&ke("WebGLRenderer: Image in DataTexture is too big ("+Q.width+"x"+Q.height+")."),T;return T}function g(T){return T.generateMipmaps}function E(T){i.generateMipmap(T)}function S(T){return T.isWebGLCubeRenderTarget?i.TEXTURE_CUBE_MAP:T.isWebGL3DRenderTarget?i.TEXTURE_3D:T.isWebGLArrayRenderTarget||T.isCompressedArrayTexture?i.TEXTURE_2D_ARRAY:i.TEXTURE_2D}function x(T,_,O,W,Q,_e=!1){if(T!==null){if(i[T]!==void 0)return i[T];ke("WebGLRenderer: Attempt to use non-existing WebGL internal format '"+T+"'")}let Se;W&&(Se=e.get("EXT_texture_norm16"),Se||ke("WebGLRenderer: Unable to use normalized textures without EXT_texture_norm16 extension"));let ae=_;if(_===i.RED&&(O===i.FLOAT&&(ae=i.R32F),O===i.HALF_FLOAT&&(ae=i.R16F),O===i.UNSIGNED_BYTE&&(ae=i.R8),O===i.UNSIGNED_SHORT&&Se&&(ae=Se.R16_EXT),O===i.SHORT&&Se&&(ae=Se.R16_SNORM_EXT)),_===i.RED_INTEGER&&(O===i.UNSIGNED_BYTE&&(ae=i.R8UI),O===i.UNSIGNED_SHORT&&(ae=i.R16UI),O===i.UNSIGNED_INT&&(ae=i.R32UI),O===i.BYTE&&(ae=i.R8I),O===i.SHORT&&(ae=i.R16I),O===i.INT&&(ae=i.R32I)),_===i.RG&&(O===i.FLOAT&&(ae=i.RG32F),O===i.HALF_FLOAT&&(ae=i.RG16F),O===i.UNSIGNED_BYTE&&(ae=i.RG8),O===i.UNSIGNED_SHORT&&Se&&(ae=Se.RG16_EXT),O===i.SHORT&&Se&&(ae=Se.RG16_SNORM_EXT)),_===i.RG_INTEGER&&(O===i.UNSIGNED_BYTE&&(ae=i.RG8UI),O===i.UNSIGNED_SHORT&&(ae=i.RG16UI),O===i.UNSIGNED_INT&&(ae=i.RG32UI),O===i.BYTE&&(ae=i.RG8I),O===i.SHORT&&(ae=i.RG16I),O===i.INT&&(ae=i.RG32I)),_===i.RGB_INTEGER&&(O===i.UNSIGNED_BYTE&&(ae=i.RGB8UI),O===i.UNSIGNED_SHORT&&(ae=i.RGB16UI),O===i.UNSIGNED_INT&&(ae=i.RGB32UI),O===i.BYTE&&(ae=i.RGB8I),O===i.SHORT&&(ae=i.RGB16I),O===i.INT&&(ae=i.RGB32I)),_===i.RGBA_INTEGER&&(O===i.UNSIGNED_BYTE&&(ae=i.RGBA8UI),O===i.UNSIGNED_SHORT&&(ae=i.RGBA16UI),O===i.UNSIGNED_INT&&(ae=i.RGBA32UI),O===i.BYTE&&(ae=i.RGBA8I),O===i.SHORT&&(ae=i.RGBA16I),O===i.INT&&(ae=i.RGBA32I)),_===i.RGB&&(O===i.UNSIGNED_SHORT&&Se&&(ae=Se.RGB16_EXT),O===i.SHORT&&Se&&(ae=Se.RGB16_SNORM_EXT),O===i.UNSIGNED_INT_5_9_9_9_REV&&(ae=i.RGB9_E5),O===i.UNSIGNED_INT_10F_11F_11F_REV&&(ae=i.R11F_G11F_B10F)),_===i.RGBA){let fe=_e?zr:Qe.getTransfer(Q);O===i.FLOAT&&(ae=i.RGBA32F),O===i.HALF_FLOAT&&(ae=i.RGBA16F),O===i.UNSIGNED_BYTE&&(ae=fe===lt?i.SRGB8_ALPHA8:i.RGBA8),O===i.UNSIGNED_SHORT&&Se&&(ae=Se.RGBA16_EXT),O===i.SHORT&&Se&&(ae=Se.RGBA16_SNORM_EXT),O===i.UNSIGNED_SHORT_4_4_4_4&&(ae=i.RGBA4),O===i.UNSIGNED_SHORT_5_5_5_1&&(ae=i.RGB5_A1)}return(ae===i.R16F||ae===i.R32F||ae===i.RG16F||ae===i.RG32F||ae===i.RGBA16F||ae===i.RGBA32F)&&e.get("EXT_color_buffer_float"),ae}function A(T,_){let O;return T?_===null||_===Wn||_===ur?O=i.DEPTH24_STENCIL8:_===nn?O=i.DEPTH32F_STENCIL8:_===hr&&(O=i.DEPTH24_STENCIL8,ke("DepthTexture: 16 bit depth attachment is not supported with stencil. Using 24-bit attachment.")):_===null||_===Wn||_===ur?O=i.DEPTH_COMPONENT24:_===nn?O=i.DEPTH_COMPONENT32F:_===hr&&(O=i.DEPTH_COMPONENT16),O}function w(T,_){return g(T)===!0||T.isFramebufferTexture&&T.minFilter!==St&&T.minFilter!==bt?Math.log2(Math.max(_.width,_.height))+1:T.mipmaps!==void 0&&T.mipmaps.length>0?T.mipmaps.length:T.isCompressedTexture&&Array.isArray(T.image)?_.mipmaps.length:1}function I(T){let _=T.target;_.removeEventListener("dispose",I),R(_),_.isVideoTexture&&h.delete(_),_.isHTMLTexture&&u.delete(_)}function v(T){let _=T.target;_.removeEventListener("dispose",v),N(_)}function R(T){let _=n.get(T);if(_.__webglInit===void 0)return;let O=T.source,W=d.get(O);if(W){let Q=W[_.__cacheKey];Q.usedTimes--,Q.usedTimes===0&&F(T),Object.keys(W).length===0&&d.delete(O)}n.remove(T)}function F(T){let _=n.get(T);i.deleteTexture(_.__webglTexture);let O=T.source,W=d.get(O);delete W[_.__cacheKey],o.memory.textures--}function N(T){let _=n.get(T);if(T.depthTexture&&(T.depthTexture.dispose(),n.remove(T.depthTexture)),T.isWebGLCubeRenderTarget)for(let W=0;W<6;W++){if(Array.isArray(_.__webglFramebuffer[W]))for(let Q=0;Q<_.__webglFramebuffer[W].length;Q++)i.deleteFramebuffer(_.__webglFramebuffer[W][Q]);else i.deleteFramebuffer(_.__webglFramebuffer[W]);_.__webglDepthbuffer&&i.deleteRenderbuffer(_.__webglDepthbuffer[W])}else{if(Array.isArray(_.__webglFramebuffer))for(let W=0;W<_.__webglFramebuffer.length;W++)i.deleteFramebuffer(_.__webglFramebuffer[W]);else i.deleteFramebuffer(_.__webglFramebuffer);if(_.__webglDepthbuffer&&i.deleteRenderbuffer(_.__webglDepthbuffer),_.__webglMultisampledFramebuffer&&i.deleteFramebuffer(_.__webglMultisampledFramebuffer),_.__webglColorRenderbuffer)for(let W=0;W<_.__webglColorRenderbuffer.length;W++)_.__webglColorRenderbuffer[W]&&i.deleteRenderbuffer(_.__webglColorRenderbuffer[W]);_.__webglDepthRenderbuffer&&i.deleteRenderbuffer(_.__webglDepthRenderbuffer)}let O=T.textures;for(let W=0,Q=O.length;W<Q;W++){let _e=n.get(O[W]);_e.__webglTexture&&(i.deleteTexture(_e.__webglTexture),o.memory.textures--),n.remove(O[W])}n.remove(T)}let H=0;function oe(){H=0}function ie(){return H}function X(T){H=T}function ee(){let T=H;return T>=s.maxTextures&&ke("WebGLTextures: Trying to use "+T+" texture units while this GPU supports only "+s.maxTextures),H+=1,T}function Y(T){let _=[];return _.push(T.wrapS),_.push(T.wrapT),_.push(T.wrapR||0),_.push(T.magFilter),_.push(T.minFilter),_.push(T.anisotropy),_.push(T.internalFormat),_.push(T.format),_.push(T.type),_.push(T.generateMipmaps),_.push(T.premultiplyAlpha),_.push(T.flipY),_.push(T.unpackAlignment),_.push(T.colorSpace),_.join()}function G(T,_){let O=n.get(T);if(T.isVideoTexture&&b(T),T.isRenderTargetTexture===!1&&T.isExternalTexture!==!0&&T.version>0&&O.__version!==T.version){let W=T.image;if(W===null)ke("WebGLRenderer: Texture marked for update but no image data found.");else if(W.complete===!1)ke("WebGLRenderer: Texture marked for update but image is incomplete");else{L(O,T,_);return}}else T.isExternalTexture&&(O.__webglTexture=T.sourceTexture?T.sourceTexture:null);t.bindTexture(i.TEXTURE_2D,O.__webglTexture,i.TEXTURE0+_)}function ge(T,_){let O=n.get(T);if(T.isRenderTargetTexture===!1&&T.version>0&&O.__version!==T.version){L(O,T,_);return}else T.isExternalTexture&&(O.__webglTexture=T.sourceTexture?T.sourceTexture:null);t.bindTexture(i.TEXTURE_2D_ARRAY,O.__webglTexture,i.TEXTURE0+_)}function ye(T,_){let O=n.get(T);if(T.isRenderTargetTexture===!1&&T.version>0&&O.__version!==T.version){L(O,T,_);return}t.bindTexture(i.TEXTURE_3D,O.__webglTexture,i.TEXTURE0+_)}function Me(T,_){let O=n.get(T);if(T.isCubeDepthTexture!==!0&&T.version>0&&O.__version!==T.version){P(O,T,_);return}t.bindTexture(i.TEXTURE_CUBE_MAP,O.__webglTexture,i.TEXTURE0+_)}let Re={[Jn]:i.REPEAT,[Gt]:i.CLAMP_TO_EDGE,[zi]:i.MIRRORED_REPEAT},ze={[St]:i.NEAREST,[Va]:i.NEAREST_MIPMAP_NEAREST,[Ss]:i.NEAREST_MIPMAP_LINEAR,[bt]:i.LINEAR,[lr]:i.LINEAR_MIPMAP_NEAREST,[yn]:i.LINEAR_MIPMAP_LINEAR},qe={[Ef]:i.NEVER,[Pf]:i.ALWAYS,[Af]:i.LESS,[Ac]:i.LEQUAL,[wf]:i.EQUAL,[wc]:i.GEQUAL,[Rf]:i.GREATER,[Cf]:i.NOTEQUAL};function We(T,_){if(_.type===nn&&e.has("OES_texture_float_linear")===!1&&(_.magFilter===bt||_.magFilter===lr||_.magFilter===Ss||_.magFilter===yn||_.minFilter===bt||_.minFilter===lr||_.minFilter===Ss||_.minFilter===yn)&&ke("WebGLRenderer: Unable to use linear filtering with floating point textures. OES_texture_float_linear not supported on this device."),i.texParameteri(T,i.TEXTURE_WRAP_S,Re[_.wrapS]),i.texParameteri(T,i.TEXTURE_WRAP_T,Re[_.wrapT]),(T===i.TEXTURE_3D||T===i.TEXTURE_2D_ARRAY)&&i.texParameteri(T,i.TEXTURE_WRAP_R,Re[_.wrapR]),i.texParameteri(T,i.TEXTURE_MAG_FILTER,ze[_.magFilter]),i.texParameteri(T,i.TEXTURE_MIN_FILTER,ze[_.minFilter]),_.compareFunction&&(i.texParameteri(T,i.TEXTURE_COMPARE_MODE,i.COMPARE_REF_TO_TEXTURE),i.texParameteri(T,i.TEXTURE_COMPARE_FUNC,qe[_.compareFunction])),e.has("EXT_texture_filter_anisotropic")===!0){if(_.magFilter===St||_.minFilter!==Ss&&_.minFilter!==yn||_.type===nn&&e.has("OES_texture_float_linear")===!1)return;if(_.anisotropy>1||n.get(_).__currentAnisotropy){let O=e.get("EXT_texture_filter_anisotropic");i.texParameterf(T,O.TEXTURE_MAX_ANISOTROPY_EXT,Math.min(_.anisotropy,s.getMaxAnisotropy())),n.get(_).__currentAnisotropy=_.anisotropy}}}function pe(T,_){let O=!1;T.__webglInit===void 0&&(T.__webglInit=!0,_.addEventListener("dispose",I));let W=_.source,Q=d.get(W);Q===void 0&&(Q={},d.set(W,Q));let _e=Y(_);if(_e!==T.__cacheKey){Q[_e]===void 0&&(Q[_e]={texture:i.createTexture(),usedTimes:0},o.memory.textures++,O=!0),Q[_e].usedTimes++;let Se=Q[T.__cacheKey];Se!==void 0&&(Q[T.__cacheKey].usedTimes--,Se.usedTimes===0&&F(_)),T.__cacheKey=_e,T.__webglTexture=Q[_e].texture}return O}function q(T,_,O){return Math.floor(Math.floor(T/O)/_)}function U(T,_,O,W){let _e=T.updateRanges;if(_e.length===0)t.texSubImage2D(i.TEXTURE_2D,0,0,0,_.width,_.height,O,W,_.data);else{_e.sort((xe,ue)=>xe.start-ue.start);let Se=0;for(let xe=1;xe<_e.length;xe++){let ue=_e[Se],ve=_e[xe],Ee=ue.start+ue.count,De=q(ve.start,_.width,4),je=q(ue.start,_.width,4);ve.start<=Ee+1&&De===je&&q(ve.start+ve.count-1,_.width,4)===De?ue.count=Math.max(ue.count,ve.start+ve.count-ue.start):(++Se,_e[Se]=ve)}_e.length=Se+1;let ae=t.getParameter(i.UNPACK_ROW_LENGTH),fe=t.getParameter(i.UNPACK_SKIP_PIXELS),J=t.getParameter(i.UNPACK_SKIP_ROWS);t.pixelStorei(i.UNPACK_ROW_LENGTH,_.width);for(let xe=0,ue=_e.length;xe<ue;xe++){let ve=_e[xe],Ee=Math.floor(ve.start/4),De=Math.ceil(ve.count/4),je=Ee%_.width,Z=Math.floor(Ee/_.width),Pe=De,be=1;t.pixelStorei(i.UNPACK_SKIP_PIXELS,je),t.pixelStorei(i.UNPACK_SKIP_ROWS,Z),t.texSubImage2D(i.TEXTURE_2D,0,je,Z,Pe,be,O,W,_.data)}T.clearUpdateRanges(),t.pixelStorei(i.UNPACK_ROW_LENGTH,ae),t.pixelStorei(i.UNPACK_SKIP_PIXELS,fe),t.pixelStorei(i.UNPACK_SKIP_ROWS,J)}}function L(T,_,O){let W=i.TEXTURE_2D;(_.isDataArrayTexture||_.isCompressedArrayTexture)&&(W=i.TEXTURE_2D_ARRAY),_.isData3DTexture&&(W=i.TEXTURE_3D);let Q=pe(T,_),_e=_.source;t.bindTexture(W,T.__webglTexture,i.TEXTURE0+O);let Se=n.get(_e);if(_e.version!==Se.__version||Q===!0){if(t.activeTexture(i.TEXTURE0+O),(typeof ImageBitmap<"u"&&_.image instanceof ImageBitmap)===!1){let be=Qe.getPrimaries(Qe.workingColorSpace),Ie=_.colorSpace===An?null:Qe.getPrimaries(_.colorSpace),Ce=_.colorSpace===An||be===Ie?i.NONE:i.BROWSER_DEFAULT_WEBGL;t.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,_.flipY),t.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,_.premultiplyAlpha),t.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,Ce)}t.pixelStorei(i.UNPACK_ALIGNMENT,_.unpackAlignment);let fe=m(_.image,!1,s.maxTextureSize);fe=we(_,fe);let J=r.convert(_.format,_.colorSpace),xe=r.convert(_.type),ue=x(_.internalFormat,J,xe,_.normalized,_.colorSpace,_.isVideoTexture);We(W,_);let ve,Ee=_.mipmaps,De=_.isVideoTexture!==!0,je=Se.__version===void 0||Q===!0,Z=_e.dataReady,Pe=w(_,fe);if(_.isDepthTexture)ue=A(_.format===ji,_.type),je&&(De?t.texStorage2D(i.TEXTURE_2D,1,ue,fe.width,fe.height):t.texImage2D(i.TEXTURE_2D,0,ue,fe.width,fe.height,0,J,xe,null));else if(_.isDataTexture)if(Ee.length>0){De&&je&&t.texStorage2D(i.TEXTURE_2D,Pe,ue,Ee[0].width,Ee[0].height);for(let be=0,Ie=Ee.length;be<Ie;be++)ve=Ee[be],De?Z&&t.texSubImage2D(i.TEXTURE_2D,be,0,0,ve.width,ve.height,J,xe,ve.data):t.texImage2D(i.TEXTURE_2D,be,ue,ve.width,ve.height,0,J,xe,ve.data);_.generateMipmaps=!1}else De?(je&&t.texStorage2D(i.TEXTURE_2D,Pe,ue,fe.width,fe.height),Z&&U(_,fe,J,xe)):t.texImage2D(i.TEXTURE_2D,0,ue,fe.width,fe.height,0,J,xe,fe.data);else if(_.isCompressedTexture)if(_.isCompressedArrayTexture){De&&je&&t.texStorage3D(i.TEXTURE_2D_ARRAY,Pe,ue,Ee[0].width,Ee[0].height,fe.depth);for(let be=0,Ie=Ee.length;be<Ie;be++)if(ve=Ee[be],_.format!==vn)if(J!==null)if(De){if(Z)if(_.layerUpdates.size>0){let Ce=nh(ve.width,ve.height,_.format,_.type);for(let Te of _.layerUpdates){let He=ve.data.subarray(Te*Ce/ve.data.BYTES_PER_ELEMENT,(Te+1)*Ce/ve.data.BYTES_PER_ELEMENT);t.compressedTexSubImage3D(i.TEXTURE_2D_ARRAY,be,0,0,Te,ve.width,ve.height,1,J,He)}_.clearLayerUpdates()}else t.compressedTexSubImage3D(i.TEXTURE_2D_ARRAY,be,0,0,0,ve.width,ve.height,fe.depth,J,ve.data)}else t.compressedTexImage3D(i.TEXTURE_2D_ARRAY,be,ue,ve.width,ve.height,fe.depth,0,ve.data,0,0);else ke("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()");else De?Z&&t.texSubImage3D(i.TEXTURE_2D_ARRAY,be,0,0,0,ve.width,ve.height,fe.depth,J,xe,ve.data):t.texImage3D(i.TEXTURE_2D_ARRAY,be,ue,ve.width,ve.height,fe.depth,0,J,xe,ve.data)}else{De&&je&&t.texStorage2D(i.TEXTURE_2D,Pe,ue,Ee[0].width,Ee[0].height);for(let be=0,Ie=Ee.length;be<Ie;be++)ve=Ee[be],_.format!==vn?J!==null?De?Z&&t.compressedTexSubImage2D(i.TEXTURE_2D,be,0,0,ve.width,ve.height,J,ve.data):t.compressedTexImage2D(i.TEXTURE_2D,be,ue,ve.width,ve.height,0,ve.data):ke("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()"):De?Z&&t.texSubImage2D(i.TEXTURE_2D,be,0,0,ve.width,ve.height,J,xe,ve.data):t.texImage2D(i.TEXTURE_2D,be,ue,ve.width,ve.height,0,J,xe,ve.data)}else if(_.isDataArrayTexture)if(De){if(je&&t.texStorage3D(i.TEXTURE_2D_ARRAY,Pe,ue,fe.width,fe.height,fe.depth),Z)if(_.layerUpdates.size>0){let be=nh(fe.width,fe.height,_.format,_.type);for(let Ie of _.layerUpdates){let Ce=fe.data.subarray(Ie*be/fe.data.BYTES_PER_ELEMENT,(Ie+1)*be/fe.data.BYTES_PER_ELEMENT);t.texSubImage3D(i.TEXTURE_2D_ARRAY,0,0,0,Ie,fe.width,fe.height,1,J,xe,Ce)}_.clearLayerUpdates()}else t.texSubImage3D(i.TEXTURE_2D_ARRAY,0,0,0,0,fe.width,fe.height,fe.depth,J,xe,fe.data)}else t.texImage3D(i.TEXTURE_2D_ARRAY,0,ue,fe.width,fe.height,fe.depth,0,J,xe,fe.data);else if(_.isData3DTexture)De?(je&&t.texStorage3D(i.TEXTURE_3D,Pe,ue,fe.width,fe.height,fe.depth),Z&&t.texSubImage3D(i.TEXTURE_3D,0,0,0,0,fe.width,fe.height,fe.depth,J,xe,fe.data)):t.texImage3D(i.TEXTURE_3D,0,ue,fe.width,fe.height,fe.depth,0,J,xe,fe.data);else if(_.isFramebufferTexture){if(je)if(De)t.texStorage2D(i.TEXTURE_2D,Pe,ue,fe.width,fe.height);else{let be=fe.width,Ie=fe.height;for(let Ce=0;Ce<Pe;Ce++)t.texImage2D(i.TEXTURE_2D,Ce,ue,be,Ie,0,J,xe,null),be>>=1,Ie>>=1}}else if(_.isHTMLTexture){if("texElementImage2D"in i){let be=i.canvas;if(be.hasAttribute("layoutsubtree")||be.setAttribute("layoutsubtree","true"),fe.parentNode!==be){be.appendChild(fe),u.add(_),be.onpaint=Ie=>{let Ce=Ie.changedElements;for(let Te of u)Ce.includes(Te.image)&&(Te.needsUpdate=!0)},be.requestPaint();return}if(i.texElementImage2D.length===3)i.texElementImage2D(i.TEXTURE_2D,i.RGBA8,fe);else{let Ce=i.RGBA,Te=i.RGBA,He=i.UNSIGNED_BYTE;i.texElementImage2D(i.TEXTURE_2D,0,Ce,Te,He,fe)}i.texParameteri(i.TEXTURE_2D,i.TEXTURE_MIN_FILTER,i.LINEAR),i.texParameteri(i.TEXTURE_2D,i.TEXTURE_WRAP_S,i.CLAMP_TO_EDGE),i.texParameteri(i.TEXTURE_2D,i.TEXTURE_WRAP_T,i.CLAMP_TO_EDGE)}}else if(Ee.length>0){if(De&&je){let be=Ae(Ee[0]);t.texStorage2D(i.TEXTURE_2D,Pe,ue,be.width,be.height)}for(let be=0,Ie=Ee.length;be<Ie;be++)ve=Ee[be],De?Z&&t.texSubImage2D(i.TEXTURE_2D,be,0,0,J,xe,ve):t.texImage2D(i.TEXTURE_2D,be,ue,J,xe,ve);_.generateMipmaps=!1}else if(De){if(je){let be=Ae(fe);t.texStorage2D(i.TEXTURE_2D,Pe,ue,be.width,be.height)}Z&&t.texSubImage2D(i.TEXTURE_2D,0,0,0,J,xe,fe)}else t.texImage2D(i.TEXTURE_2D,0,ue,J,xe,fe);g(_)&&E(W),Se.__version=_e.version,_.onUpdate&&_.onUpdate(_)}T.__version=_.version}function P(T,_,O){if(_.image.length!==6)return;let W=pe(T,_),Q=_.source;t.bindTexture(i.TEXTURE_CUBE_MAP,T.__webglTexture,i.TEXTURE0+O);let _e=n.get(Q);if(Q.version!==_e.__version||W===!0){t.activeTexture(i.TEXTURE0+O);let Se=Qe.getPrimaries(Qe.workingColorSpace),ae=_.colorSpace===An?null:Qe.getPrimaries(_.colorSpace),fe=_.colorSpace===An||Se===ae?i.NONE:i.BROWSER_DEFAULT_WEBGL;t.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,_.flipY),t.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,_.premultiplyAlpha),t.pixelStorei(i.UNPACK_ALIGNMENT,_.unpackAlignment),t.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,fe);let J=_.isCompressedTexture||_.image[0].isCompressedTexture,xe=_.image[0]&&_.image[0].isDataTexture,ue=[];for(let Te=0;Te<6;Te++)!J&&!xe?ue[Te]=m(_.image[Te],!0,s.maxCubemapSize):ue[Te]=xe?_.image[Te].image:_.image[Te],ue[Te]=we(_,ue[Te]);let ve=ue[0],Ee=r.convert(_.format,_.colorSpace),De=r.convert(_.type),je=x(_.internalFormat,Ee,De,_.normalized,_.colorSpace),Z=_.isVideoTexture!==!0,Pe=_e.__version===void 0||W===!0,be=Q.dataReady,Ie=w(_,ve);We(i.TEXTURE_CUBE_MAP,_);let Ce;if(J){Z&&Pe&&t.texStorage2D(i.TEXTURE_CUBE_MAP,Ie,je,ve.width,ve.height);for(let Te=0;Te<6;Te++){Ce=ue[Te].mipmaps;for(let He=0;He<Ce.length;He++){let Be=Ce[He];_.format!==vn?Ee!==null?Z?be&&t.compressedTexSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He,0,0,Be.width,Be.height,Ee,Be.data):t.compressedTexImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He,je,Be.width,Be.height,0,Be.data):ke("WebGLRenderer: Attempt to load unsupported compressed texture format in .setTextureCube()"):Z?be&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He,0,0,Be.width,Be.height,Ee,De,Be.data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He,je,Be.width,Be.height,0,Ee,De,Be.data)}}}else{if(Ce=_.mipmaps,Z&&Pe){Ce.length>0&&Ie++;let Te=Ae(ue[0]);t.texStorage2D(i.TEXTURE_CUBE_MAP,Ie,je,Te.width,Te.height)}for(let Te=0;Te<6;Te++)if(xe){Z?be&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,0,0,ue[Te].width,ue[Te].height,Ee,De,ue[Te].data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,je,ue[Te].width,ue[Te].height,0,Ee,De,ue[Te].data);for(let He=0;He<Ce.length;He++){let dt=Ce[He].image[Te].image;Z?be&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He+1,0,0,dt.width,dt.height,Ee,De,dt.data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He+1,je,dt.width,dt.height,0,Ee,De,dt.data)}}else{Z?be&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,0,0,Ee,De,ue[Te]):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,je,Ee,De,ue[Te]);for(let He=0;He<Ce.length;He++){let Be=Ce[He];Z?be&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He+1,0,0,Ee,De,Be.image[Te]):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+Te,He+1,je,Ee,De,Be.image[Te])}}}g(_)&&E(i.TEXTURE_CUBE_MAP),_e.__version=Q.version,_.onUpdate&&_.onUpdate(_)}T.__version=_.version}function z(T,_,O,W,Q,_e){let Se=r.convert(O.format,O.colorSpace),ae=r.convert(O.type),fe=x(O.internalFormat,Se,ae,O.normalized,O.colorSpace),J=n.get(_),xe=n.get(O);if(xe.__renderTarget=_,!J.__hasExternalTextures){let ue=Math.max(1,_.width>>_e),ve=Math.max(1,_.height>>_e);Q===i.TEXTURE_3D||Q===i.TEXTURE_2D_ARRAY?t.texImage3D(Q,_e,fe,ue,ve,_.depth,0,Se,ae,null):t.texImage2D(Q,_e,fe,ue,ve,0,Se,ae,null)}t.bindFramebuffer(i.FRAMEBUFFER,T),k(_)?a.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,W,Q,xe.__webglTexture,0,ne(_)):(Q===i.TEXTURE_2D||Q>=i.TEXTURE_CUBE_MAP_POSITIVE_X&&Q<=i.TEXTURE_CUBE_MAP_NEGATIVE_Z)&&i.framebufferTexture2D(i.FRAMEBUFFER,W,Q,xe.__webglTexture,_e),t.bindFramebuffer(i.FRAMEBUFFER,null)}function he(T,_,O){if(i.bindRenderbuffer(i.RENDERBUFFER,T),_.depthBuffer){let W=_.depthTexture,Q=W&&W.isDepthTexture?W.type:null,_e=A(_.stencilBuffer,Q),Se=_.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;k(_)?a.renderbufferStorageMultisampleEXT(i.RENDERBUFFER,ne(_),_e,_.width,_.height):O?i.renderbufferStorageMultisample(i.RENDERBUFFER,ne(_),_e,_.width,_.height):i.renderbufferStorage(i.RENDERBUFFER,_e,_.width,_.height),i.framebufferRenderbuffer(i.FRAMEBUFFER,Se,i.RENDERBUFFER,T)}else{let W=_.textures;for(let Q=0;Q<W.length;Q++){let _e=W[Q],Se=r.convert(_e.format,_e.colorSpace),ae=r.convert(_e.type),fe=x(_e.internalFormat,Se,ae,_e.normalized,_e.colorSpace);k(_)?a.renderbufferStorageMultisampleEXT(i.RENDERBUFFER,ne(_),fe,_.width,_.height):O?i.renderbufferStorageMultisample(i.RENDERBUFFER,ne(_),fe,_.width,_.height):i.renderbufferStorage(i.RENDERBUFFER,fe,_.width,_.height)}}i.bindRenderbuffer(i.RENDERBUFFER,null)}function me(T,_,O){let W=_.isWebGLCubeRenderTarget===!0;if(t.bindFramebuffer(i.FRAMEBUFFER,T),!(_.depthTexture&&_.depthTexture.isDepthTexture))throw new Error("THREE.WebGLTextures: renderTarget.depthTexture must be an instance of THREE.DepthTexture.");let Q=n.get(_.depthTexture);if(Q.__renderTarget=_,(!Q.__webglTexture||_.depthTexture.image.width!==_.width||_.depthTexture.image.height!==_.height)&&(_.depthTexture.image.width=_.width,_.depthTexture.image.height=_.height,_.depthTexture.needsUpdate=!0),W){if(Q.__webglInit===void 0&&(Q.__webglInit=!0,_.depthTexture.addEventListener("dispose",I)),Q.__webglTexture===void 0){Q.__webglTexture=i.createTexture(),t.bindTexture(i.TEXTURE_CUBE_MAP,Q.__webglTexture),We(i.TEXTURE_CUBE_MAP,_.depthTexture);let J=r.convert(_.depthTexture.format),xe=r.convert(_.depthTexture.type),ue;_.depthTexture.format===$n?ue=i.DEPTH_COMPONENT24:_.depthTexture.format===ji&&(ue=i.DEPTH24_STENCIL8);for(let ve=0;ve<6;ve++)i.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ve,0,ue,_.width,_.height,0,J,xe,null)}}else G(_.depthTexture,0);let _e=Q.__webglTexture,Se=ne(_),ae=W?i.TEXTURE_CUBE_MAP_POSITIVE_X+O:i.TEXTURE_2D,fe=_.depthTexture.format===ji?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;if(_.depthTexture.format===$n)k(_)?a.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,fe,ae,_e,0,Se):i.framebufferTexture2D(i.FRAMEBUFFER,fe,ae,_e,0);else if(_.depthTexture.format===ji)k(_)?a.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,fe,ae,_e,0,Se):i.framebufferTexture2D(i.FRAMEBUFFER,fe,ae,_e,0);else throw new Error("THREE.WebGLTextures: Unknown depthTexture format.")}function C(T){let _=n.get(T),O=T.isWebGLCubeRenderTarget===!0;if(_.__boundDepthTexture!==T.depthTexture){let W=T.depthTexture;if(_.__depthDisposeCallback&&_.__depthDisposeCallback(),W){let Q=()=>{delete _.__boundDepthTexture,delete _.__depthDisposeCallback,W.removeEventListener("dispose",Q)};W.addEventListener("dispose",Q),_.__depthDisposeCallback=Q}_.__boundDepthTexture=W}if(T.depthTexture&&!_.__autoAllocateDepthBuffer)if(O)for(let W=0;W<6;W++)me(_.__webglFramebuffer[W],T,W);else{let W=T.texture.mipmaps;W&&W.length>0?me(_.__webglFramebuffer[0],T,0):me(_.__webglFramebuffer,T,0)}else if(O){_.__webglDepthbuffer=[];for(let W=0;W<6;W++)if(t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer[W]),_.__webglDepthbuffer[W]===void 0)_.__webglDepthbuffer[W]=i.createRenderbuffer(),he(_.__webglDepthbuffer[W],T,!1);else{let Q=T.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,_e=_.__webglDepthbuffer[W];i.bindRenderbuffer(i.RENDERBUFFER,_e),i.framebufferRenderbuffer(i.FRAMEBUFFER,Q,i.RENDERBUFFER,_e)}}else{let W=T.texture.mipmaps;if(W&&W.length>0?t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer[0]):t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer),_.__webglDepthbuffer===void 0)_.__webglDepthbuffer=i.createRenderbuffer(),he(_.__webglDepthbuffer,T,!1);else{let Q=T.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,_e=_.__webglDepthbuffer;i.bindRenderbuffer(i.RENDERBUFFER,_e),i.framebufferRenderbuffer(i.FRAMEBUFFER,Q,i.RENDERBUFFER,_e)}}t.bindFramebuffer(i.FRAMEBUFFER,null)}function V(T,_,O){let W=n.get(T);_!==void 0&&z(W.__webglFramebuffer,T,T.texture,i.COLOR_ATTACHMENT0,i.TEXTURE_2D,0),O!==void 0&&C(T)}function D(T){let _=T.texture,O=n.get(T),W=n.get(_);T.addEventListener("dispose",v);let Q=T.textures,_e=T.isWebGLCubeRenderTarget===!0,Se=Q.length>1;if(Se||(W.__webglTexture===void 0&&(W.__webglTexture=i.createTexture()),W.__version=_.version,o.memory.textures++),_e){O.__webglFramebuffer=[];for(let ae=0;ae<6;ae++)if(_.mipmaps&&_.mipmaps.length>0){O.__webglFramebuffer[ae]=[];for(let fe=0;fe<_.mipmaps.length;fe++)O.__webglFramebuffer[ae][fe]=i.createFramebuffer()}else O.__webglFramebuffer[ae]=i.createFramebuffer()}else{if(_.mipmaps&&_.mipmaps.length>0){O.__webglFramebuffer=[];for(let ae=0;ae<_.mipmaps.length;ae++)O.__webglFramebuffer[ae]=i.createFramebuffer()}else O.__webglFramebuffer=i.createFramebuffer();if(Se)for(let ae=0,fe=Q.length;ae<fe;ae++){let J=n.get(Q[ae]);J.__webglTexture===void 0&&(J.__webglTexture=i.createTexture(),o.memory.textures++)}if(T.samples>0&&k(T)===!1){O.__webglMultisampledFramebuffer=i.createFramebuffer(),O.__webglColorRenderbuffer=[],t.bindFramebuffer(i.FRAMEBUFFER,O.__webglMultisampledFramebuffer);for(let ae=0;ae<Q.length;ae++){let fe=Q[ae];O.__webglColorRenderbuffer[ae]=i.createRenderbuffer(),i.bindRenderbuffer(i.RENDERBUFFER,O.__webglColorRenderbuffer[ae]);let J=r.convert(fe.format,fe.colorSpace),xe=r.convert(fe.type),ue=x(fe.internalFormat,J,xe,fe.normalized,fe.colorSpace,T.isXRRenderTarget===!0),ve=ne(T);i.renderbufferStorageMultisample(i.RENDERBUFFER,ve,ue,T.width,T.height),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+ae,i.RENDERBUFFER,O.__webglColorRenderbuffer[ae])}i.bindRenderbuffer(i.RENDERBUFFER,null),T.depthBuffer&&(O.__webglDepthRenderbuffer=i.createRenderbuffer(),he(O.__webglDepthRenderbuffer,T,!0)),t.bindFramebuffer(i.FRAMEBUFFER,null)}}if(_e){t.bindTexture(i.TEXTURE_CUBE_MAP,W.__webglTexture),We(i.TEXTURE_CUBE_MAP,_);for(let ae=0;ae<6;ae++)if(_.mipmaps&&_.mipmaps.length>0)for(let fe=0;fe<_.mipmaps.length;fe++)z(O.__webglFramebuffer[ae][fe],T,_,i.COLOR_ATTACHMENT0,i.TEXTURE_CUBE_MAP_POSITIVE_X+ae,fe);else z(O.__webglFramebuffer[ae],T,_,i.COLOR_ATTACHMENT0,i.TEXTURE_CUBE_MAP_POSITIVE_X+ae,0);g(_)&&E(i.TEXTURE_CUBE_MAP),t.unbindTexture()}else if(Se){for(let ae=0,fe=Q.length;ae<fe;ae++){let J=Q[ae],xe=n.get(J),ue=i.TEXTURE_2D;(T.isWebGL3DRenderTarget||T.isWebGLArrayRenderTarget)&&(ue=T.isWebGL3DRenderTarget?i.TEXTURE_3D:i.TEXTURE_2D_ARRAY),t.bindTexture(ue,xe.__webglTexture),We(ue,J),z(O.__webglFramebuffer,T,J,i.COLOR_ATTACHMENT0+ae,ue,0),g(J)&&E(ue)}t.unbindTexture()}else{let ae=i.TEXTURE_2D;if((T.isWebGL3DRenderTarget||T.isWebGLArrayRenderTarget)&&(ae=T.isWebGL3DRenderTarget?i.TEXTURE_3D:i.TEXTURE_2D_ARRAY),t.bindTexture(ae,W.__webglTexture),We(ae,_),_.mipmaps&&_.mipmaps.length>0)for(let fe=0;fe<_.mipmaps.length;fe++)z(O.__webglFramebuffer[fe],T,_,i.COLOR_ATTACHMENT0,ae,fe);else z(O.__webglFramebuffer,T,_,i.COLOR_ATTACHMENT0,ae,0);g(_)&&E(ae),t.unbindTexture()}T.depthBuffer&&C(T)}function j(T){let _=T.textures;for(let O=0,W=_.length;O<W;O++){let Q=_[O];if(g(Q)){let _e=S(T),Se=n.get(Q).__webglTexture;t.bindTexture(_e,Se),E(_e),t.unbindTexture()}}}let $=[],le=[];function te(T){if(T.samples>0){if(k(T)===!1){let _=T.textures,O=T.width,W=T.height,Q=i.COLOR_BUFFER_BIT,_e=T.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,Se=n.get(T),ae=_.length>1;if(ae)for(let J=0;J<_.length;J++)t.bindFramebuffer(i.FRAMEBUFFER,Se.__webglMultisampledFramebuffer),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+J,i.RENDERBUFFER,null),t.bindFramebuffer(i.FRAMEBUFFER,Se.__webglFramebuffer),i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0+J,i.TEXTURE_2D,null,0);t.bindFramebuffer(i.READ_FRAMEBUFFER,Se.__webglMultisampledFramebuffer);let fe=T.texture.mipmaps;fe&&fe.length>0?t.bindFramebuffer(i.DRAW_FRAMEBUFFER,Se.__webglFramebuffer[0]):t.bindFramebuffer(i.DRAW_FRAMEBUFFER,Se.__webglFramebuffer);for(let J=0;J<_.length;J++){if(T.resolveDepthBuffer&&(T.depthBuffer&&(Q|=i.DEPTH_BUFFER_BIT),T.stencilBuffer&&T.resolveStencilBuffer&&(Q|=i.STENCIL_BUFFER_BIT)),ae){i.framebufferRenderbuffer(i.READ_FRAMEBUFFER,i.COLOR_ATTACHMENT0,i.RENDERBUFFER,Se.__webglColorRenderbuffer[J]);let xe=n.get(_[J]).__webglTexture;i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0,i.TEXTURE_2D,xe,0)}i.blitFramebuffer(0,0,O,W,0,0,O,W,Q,i.NEAREST),c===!0&&($.length=0,le.length=0,$.push(i.COLOR_ATTACHMENT0+J),T.depthBuffer&&T.resolveDepthBuffer===!1&&($.push(_e),le.push(_e),i.invalidateFramebuffer(i.DRAW_FRAMEBUFFER,le)),i.invalidateFramebuffer(i.READ_FRAMEBUFFER,$))}if(t.bindFramebuffer(i.READ_FRAMEBUFFER,null),t.bindFramebuffer(i.DRAW_FRAMEBUFFER,null),ae)for(let J=0;J<_.length;J++){t.bindFramebuffer(i.FRAMEBUFFER,Se.__webglMultisampledFramebuffer),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+J,i.RENDERBUFFER,Se.__webglColorRenderbuffer[J]);let xe=n.get(_[J]).__webglTexture;t.bindFramebuffer(i.FRAMEBUFFER,Se.__webglFramebuffer),i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0+J,i.TEXTURE_2D,xe,0)}t.bindFramebuffer(i.DRAW_FRAMEBUFFER,Se.__webglMultisampledFramebuffer)}else if(T.depthBuffer&&T.resolveDepthBuffer===!1&&c){let _=T.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;i.invalidateFramebuffer(i.DRAW_FRAMEBUFFER,[_])}}}function ne(T){return Math.min(s.maxSamples,T.samples)}function k(T){let _=n.get(T);return T.samples>0&&e.has("WEBGL_multisampled_render_to_texture")===!0&&_.__useRenderToTexture!==!1}function b(T){let _=o.render.frame;h.get(T)!==_&&(h.set(T,_),T.update())}function we(T,_){let O=T.colorSpace,W=T.format,Q=T.type;return T.isCompressedTexture===!0||T.isVideoTexture===!0||O!==Wt&&O!==An&&(Qe.getTransfer(O)===lt?(W!==vn||Q!==hn)&&ke("WebGLTextures: sRGB encoded textures have to use RGBAFormat and UnsignedByteType."):Ke("WebGLTextures: Unsupported texture color space:",O)),_}function Ae(T){return typeof HTMLImageElement<"u"&&T instanceof HTMLImageElement?(l.width=T.naturalWidth||T.width,l.height=T.naturalHeight||T.height):typeof VideoFrame<"u"&&T instanceof VideoFrame?(l.width=T.displayWidth,l.height=T.displayHeight):(l.width=T.width,l.height=T.height),l}this.allocateTextureUnit=ee,this.resetTextureUnits=oe,this.getTextureUnits=ie,this.setTextureUnits=X,this.setTexture2D=G,this.setTexture2DArray=ge,this.setTexture3D=ye,this.setTextureCube=Me,this.rebindTextures=V,this.setupRenderTarget=D,this.updateRenderTargetMipmap=j,this.updateMultisampleRenderTarget=te,this.setupDepthRenderbuffer=C,this.setupFrameBufferTexture=z,this.useMultisampledRTT=k,this.isReversedDepthBuffer=function(){return t.buffers.depth.getReversed()}}function cy(i,e){function t(n,s=An){let r,o=Qe.getTransfer(s);if(n===hn)return i.UNSIGNED_BYTE;if(n===Wa)return i.UNSIGNED_SHORT_4_4_4_4;if(n===Xa)return i.UNSIGNED_SHORT_5_5_5_1;if(n===ql)return i.UNSIGNED_INT_5_9_9_9_REV;if(n===Yl)return i.UNSIGNED_INT_10F_11F_11F_REV;if(n===Wl)return i.BYTE;if(n===Xl)return i.SHORT;if(n===hr)return i.UNSIGNED_SHORT;if(n===Ga)return i.INT;if(n===Wn)return i.UNSIGNED_INT;if(n===nn)return i.FLOAT;if(n===ni)return i.HALF_FLOAT;if(n===Zl)return i.ALPHA;if(n===Kl)return i.RGB;if(n===vn)return i.RGBA;if(n===$n)return i.DEPTH_COMPONENT;if(n===ji)return i.DEPTH_STENCIL;if(n===fr)return i.RED;if(n===qa)return i.RED_INTEGER;if(n===Ji)return i.RG;if(n===Ya)return i.RG_INTEGER;if(n===Za)return i.RGBA_INTEGER;if(n===_o||n===xo||n===yo||n===vo)if(o===lt)if(r=e.get("WEBGL_compressed_texture_s3tc_srgb"),r!==null){if(n===_o)return r.COMPRESSED_SRGB_S3TC_DXT1_EXT;if(n===xo)return r.COMPRESSED_SRGB_ALPHA_S3TC_DXT1_EXT;if(n===yo)return r.COMPRESSED_SRGB_ALPHA_S3TC_DXT3_EXT;if(n===vo)return r.COMPRESSED_SRGB_ALPHA_S3TC_DXT5_EXT}else return null;else if(r=e.get("WEBGL_compressed_texture_s3tc"),r!==null){if(n===_o)return r.COMPRESSED_RGB_S3TC_DXT1_EXT;if(n===xo)return r.COMPRESSED_RGBA_S3TC_DXT1_EXT;if(n===yo)return r.COMPRESSED_RGBA_S3TC_DXT3_EXT;if(n===vo)return r.COMPRESSED_RGBA_S3TC_DXT5_EXT}else return null;if(n===Ka||n===ja||n===Ja||n===$a)if(r=e.get("WEBGL_compressed_texture_pvrtc"),r!==null){if(n===Ka)return r.COMPRESSED_RGB_PVRTC_4BPPV1_IMG;if(n===ja)return r.COMPRESSED_RGB_PVRTC_2BPPV1_IMG;if(n===Ja)return r.COMPRESSED_RGBA_PVRTC_4BPPV1_IMG;if(n===$a)return r.COMPRESSED_RGBA_PVRTC_2BPPV1_IMG}else return null;if(n===Qa||n===ec||n===tc||n===nc||n===ic||n===bo||n===sc)if(r=e.get("WEBGL_compressed_texture_etc"),r!==null){if(n===Qa||n===ec)return o===lt?r.COMPRESSED_SRGB8_ETC2:r.COMPRESSED_RGB8_ETC2;if(n===tc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ETC2_EAC:r.COMPRESSED_RGBA8_ETC2_EAC;if(n===nc)return r.COMPRESSED_R11_EAC;if(n===ic)return r.COMPRESSED_SIGNED_R11_EAC;if(n===bo)return r.COMPRESSED_RG11_EAC;if(n===sc)return r.COMPRESSED_SIGNED_RG11_EAC}else return null;if(n===rc||n===oc||n===ac||n===cc||n===lc||n===hc||n===uc||n===fc||n===dc||n===pc||n===mc||n===gc||n===_c||n===xc)if(r=e.get("WEBGL_compressed_texture_astc"),r!==null){if(n===rc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR:r.COMPRESSED_RGBA_ASTC_4x4_KHR;if(n===oc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR:r.COMPRESSED_RGBA_ASTC_5x4_KHR;if(n===ac)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR:r.COMPRESSED_RGBA_ASTC_5x5_KHR;if(n===cc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR:r.COMPRESSED_RGBA_ASTC_6x5_KHR;if(n===lc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR:r.COMPRESSED_RGBA_ASTC_6x6_KHR;if(n===hc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR:r.COMPRESSED_RGBA_ASTC_8x5_KHR;if(n===uc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR:r.COMPRESSED_RGBA_ASTC_8x6_KHR;if(n===fc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR:r.COMPRESSED_RGBA_ASTC_8x8_KHR;if(n===dc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR:r.COMPRESSED_RGBA_ASTC_10x5_KHR;if(n===pc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR:r.COMPRESSED_RGBA_ASTC_10x6_KHR;if(n===mc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR:r.COMPRESSED_RGBA_ASTC_10x8_KHR;if(n===gc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR:r.COMPRESSED_RGBA_ASTC_10x10_KHR;if(n===_c)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR:r.COMPRESSED_RGBA_ASTC_12x10_KHR;if(n===xc)return o===lt?r.COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR:r.COMPRESSED_RGBA_ASTC_12x12_KHR}else return null;if(n===yc||n===vc||n===bc)if(r=e.get("EXT_texture_compression_bptc"),r!==null){if(n===yc)return o===lt?r.COMPRESSED_SRGB_ALPHA_BPTC_UNORM_EXT:r.COMPRESSED_RGBA_BPTC_UNORM_EXT;if(n===vc)return r.COMPRESSED_RGB_BPTC_SIGNED_FLOAT_EXT;if(n===bc)return r.COMPRESSED_RGB_BPTC_UNSIGNED_FLOAT_EXT}else return null;if(n===Mc||n===Sc||n===Mo||n===Tc)if(r=e.get("EXT_texture_compression_rgtc"),r!==null){if(n===Mc)return r.COMPRESSED_RED_RGTC1_EXT;if(n===Sc)return r.COMPRESSED_SIGNED_RED_RGTC1_EXT;if(n===Mo)return r.COMPRESSED_RED_GREEN_RGTC2_EXT;if(n===Tc)return r.COMPRESSED_SIGNED_RED_GREEN_RGTC2_EXT}else return null;return n===ur?i.UNSIGNED_INT_24_8:i[n]!==void 0?i[n]:null}return{convert:t}}var ly=`
void main() {

	gl_Position = vec4( position, 1.0 );

}`,hy=`
uniform sampler2DArray depthColor;
uniform float depthWidth;
uniform float depthHeight;

void main() {

	vec2 coord = vec2( gl_FragCoord.x / depthWidth, gl_FragCoord.y / depthHeight );

	if ( coord.x >= 1.0 ) {

		gl_FragDepth = texture( depthColor, vec3( coord.x - 1.0, coord.y, 1 ) ).r;

	} else {

		gl_FragDepth = texture( depthColor, vec3( coord.x, coord.y, 0 ) ).r;

	}

}`,bh=class{constructor(){this.texture=null,this.mesh=null,this.depthNear=0,this.depthFar=0}init(e,t){if(this.texture===null){let n=new Qr(e.texture);(e.depthNear!==t.depthNear||e.depthFar!==t.depthFar)&&(this.depthNear=e.depthNear,this.depthFar=e.depthFar),this.texture=n}}getMesh(e){if(this.texture!==null&&this.mesh===null){let t=e.cameras[0].viewport,n=new Jt({vertexShader:ly,fragmentShader:hy,uniforms:{depthColor:{value:this.texture},depthWidth:{value:t.z},depthHeight:{value:t.w}}});this.mesh=new ht(new xs(20,20),n)}return this.mesh}reset(){this.texture=null,this.mesh=null}getDepthTexture(){return this.texture}},Mh=class extends Fn{constructor(e,t){super();let n=this,s=null,r=1,o=null,a="local-floor",c=1,l=null,h=null,u=null,f=null,d=null,p=null,y=typeof XRWebGLBinding<"u",m=new bh,g={},E=t.getContextAttributes(),S=null,x=null,A=[],w=[],I=new de,v=null,R=new Ct;R.viewport=new ft;let F=new Ct;F.viewport=new ft;let N=[R,F],H=new ka,oe=null,ie=null;this.cameraAutoUpdate=!0,this.enabled=!1,this.isPresenting=!1,this.getController=function(pe){let q=A[pe];return q===void 0&&(q=new $s,A[pe]=q),q.getTargetRaySpace()},this.getControllerGrip=function(pe){let q=A[pe];return q===void 0&&(q=new $s,A[pe]=q),q.getGripSpace()},this.getHand=function(pe){let q=A[pe];return q===void 0&&(q=new $s,A[pe]=q),q.getHandSpace()};function X(pe){let q=w.indexOf(pe.inputSource);if(q===-1)return;let U=A[q];U!==void 0&&(U.update(pe.inputSource,pe.frame,l||o),U.dispatchEvent({type:pe.type,data:pe.inputSource}))}function ee(){s.removeEventListener("select",X),s.removeEventListener("selectstart",X),s.removeEventListener("selectend",X),s.removeEventListener("squeeze",X),s.removeEventListener("squeezestart",X),s.removeEventListener("squeezeend",X),s.removeEventListener("end",ee),s.removeEventListener("inputsourceschange",Y);for(let pe=0;pe<A.length;pe++){let q=w[pe];q!==null&&(w[pe]=null,A[pe].disconnect(q))}oe=null,ie=null,m.reset();for(let pe in g)delete g[pe];e.setRenderTarget(S),d=null,f=null,u=null,s=null,x=null,We.stop(),n.isPresenting=!1,e.setPixelRatio(v),e.setSize(I.width,I.height,!1),n.dispatchEvent({type:"sessionend"})}this.setFramebufferScaleFactor=function(pe){r=pe,n.isPresenting===!0&&ke("WebXRManager: Cannot change framebuffer scale while presenting.")},this.setReferenceSpaceType=function(pe){a=pe,n.isPresenting===!0&&ke("WebXRManager: Cannot change reference space type while presenting.")},this.getReferenceSpace=function(){return l||o},this.setReferenceSpace=function(pe){l=pe},this.getBaseLayer=function(){return f!==null?f:d},this.getBinding=function(){return u===null&&y&&(u=new XRWebGLBinding(s,t)),u},this.getFrame=function(){return p},this.getSession=function(){return s},this.setSession=async function(pe){if(s=pe,s!==null){if(S=e.getRenderTarget(),s.addEventListener("select",X),s.addEventListener("selectstart",X),s.addEventListener("selectend",X),s.addEventListener("squeeze",X),s.addEventListener("squeezestart",X),s.addEventListener("squeezeend",X),s.addEventListener("end",ee),s.addEventListener("inputsourceschange",Y),E.xrCompatible!==!0&&await t.makeXRCompatible(),v=e.getPixelRatio(),e.getSize(I),y&&"createProjectionLayer"in XRWebGLBinding.prototype){let U=null,L=null,P=null;E.depth&&(P=E.stencil?t.DEPTH24_STENCIL8:t.DEPTH_COMPONENT24,U=E.stencil?ji:$n,L=E.stencil?ur:Wn);let z={colorFormat:t.RGBA8,depthFormat:P,scaleFactor:r};u=this.getBinding(),f=u.createProjectionLayer(z),s.updateRenderState({layers:[f]}),e.setPixelRatio(1),e.setSize(f.textureWidth,f.textureHeight,!1),x=new jt(f.textureWidth,f.textureHeight,{format:vn,type:hn,depthTexture:new xi(f.textureWidth,f.textureHeight,L,void 0,void 0,void 0,void 0,void 0,void 0,U),stencilBuffer:E.stencil,colorSpace:e.outputColorSpace,samples:E.antialias?4:0,resolveDepthBuffer:f.ignoreDepthValues===!1,resolveStencilBuffer:f.ignoreDepthValues===!1})}else{let U={antialias:E.antialias,alpha:!0,depth:E.depth,stencil:E.stencil,framebufferScaleFactor:r};d=new XRWebGLLayer(s,t,U),s.updateRenderState({baseLayer:d}),e.setPixelRatio(1),e.setSize(d.framebufferWidth,d.framebufferHeight,!1),x=new jt(d.framebufferWidth,d.framebufferHeight,{format:vn,type:hn,colorSpace:e.outputColorSpace,stencilBuffer:E.stencil,resolveDepthBuffer:d.ignoreDepthValues===!1,resolveStencilBuffer:d.ignoreDepthValues===!1})}x.isXRRenderTarget=!0,this.setFoveation(c),l=null,o=await s.requestReferenceSpace(a),We.setContext(s),We.start(),n.isPresenting=!0,n.dispatchEvent({type:"sessionstart"})}},this.getEnvironmentBlendMode=function(){if(s!==null)return s.environmentBlendMode},this.getDepthTexture=function(){return m.getDepthTexture()};function Y(pe){for(let q=0;q<pe.removed.length;q++){let U=pe.removed[q],L=w.indexOf(U);L>=0&&(w[L]=null,A[L].disconnect(U))}for(let q=0;q<pe.added.length;q++){let U=pe.added[q],L=w.indexOf(U);if(L===-1){for(let z=0;z<A.length;z++)if(z>=w.length){w.push(U),L=z;break}else if(w[z]===null){w[z]=U,L=z;break}if(L===-1)break}let P=A[L];P&&P.connect(U)}}let G=new B,ge=new B;function ye(pe,q,U){G.setFromMatrixPosition(q.matrixWorld),ge.setFromMatrixPosition(U.matrixWorld);let L=G.distanceTo(ge),P=q.projectionMatrix.elements,z=U.projectionMatrix.elements,he=P[14]/(P[10]-1),me=P[14]/(P[10]+1),C=(P[9]+1)/P[5],V=(P[9]-1)/P[5],D=(P[8]-1)/P[0],j=(z[8]+1)/z[0],$=he*D,le=he*j,te=L/(-D+j),ne=te*-D;if(q.matrixWorld.decompose(pe.position,pe.quaternion,pe.scale),pe.translateX(ne),pe.translateZ(te),pe.matrixWorld.compose(pe.position,pe.quaternion,pe.scale),pe.matrixWorldInverse.copy(pe.matrixWorld).invert(),P[10]===-1)pe.projectionMatrix.copy(q.projectionMatrix),pe.projectionMatrixInverse.copy(q.projectionMatrixInverse);else{let k=he+te,b=me+te,we=$-ne,Ae=le+(L-ne),T=C*me/b*k,_=V*me/b*k;pe.projectionMatrix.makePerspective(we,Ae,T,_,k,b),pe.projectionMatrixInverse.copy(pe.projectionMatrix).invert()}}function Me(pe,q){q===null?pe.matrixWorld.copy(pe.matrix):pe.matrixWorld.multiplyMatrices(q.matrixWorld,pe.matrix),pe.matrixWorldInverse.copy(pe.matrixWorld).invert()}this.updateCamera=function(pe){if(s===null)return;let q=pe.near,U=pe.far;m.texture!==null&&(m.depthNear>0&&(q=m.depthNear),m.depthFar>0&&(U=m.depthFar)),H.near=F.near=R.near=q,H.far=F.far=R.far=U,(oe!==H.near||ie!==H.far)&&(s.updateRenderState({depthNear:H.near,depthFar:H.far}),oe=H.near,ie=H.far),H.layers.mask=pe.layers.mask|6,R.layers.mask=H.layers.mask&-5,F.layers.mask=H.layers.mask&-3;let L=pe.parent,P=H.cameras;Me(H,L);for(let z=0;z<P.length;z++)Me(P[z],L);P.length===2?ye(H,R,F):H.projectionMatrix.copy(R.projectionMatrix),Re(pe,H,L)};function Re(pe,q,U){U===null?pe.matrix.copy(q.matrixWorld):(pe.matrix.copy(U.matrixWorld),pe.matrix.invert(),pe.matrix.multiply(q.matrixWorld)),pe.matrix.decompose(pe.position,pe.quaternion,pe.scale),pe.updateMatrixWorld(!0),pe.projectionMatrix.copy(q.projectionMatrix),pe.projectionMatrixInverse.copy(q.projectionMatrixInverse),pe.isPerspectiveCamera&&(pe.fov=ps*2*Math.atan(1/pe.projectionMatrix.elements[5]),pe.zoom=1)}this.getCamera=function(){return H},this.getFoveation=function(){if(!(f===null&&d===null))return c},this.setFoveation=function(pe){c=pe,f!==null&&(f.fixedFoveation=pe),d!==null&&d.fixedFoveation!==void 0&&(d.fixedFoveation=pe)},this.hasDepthSensing=function(){return m.texture!==null},this.getDepthSensingMesh=function(){return m.getMesh(H)},this.getCameraTexture=function(pe){return g[pe]};let ze=null;function qe(pe,q){if(h=q.getViewerPose(l||o),p=q,h!==null){let U=h.views;d!==null&&(e.setRenderTargetFramebuffer(x,d.framebuffer),e.setRenderTarget(x));let L=!1;U.length!==H.cameras.length&&(H.cameras.length=0,L=!0);for(let me=0;me<U.length;me++){let C=U[me],V=null;if(d!==null)V=d.getViewport(C);else{let j=u.getViewSubImage(f,C);V=j.viewport,me===0&&(e.setRenderTargetTextures(x,j.colorTexture,j.depthStencilTexture),e.setRenderTarget(x))}let D=N[me];D===void 0&&(D=new Ct,D.layers.enable(me),D.viewport=new ft,N[me]=D),D.matrix.fromArray(C.transform.matrix),D.matrix.decompose(D.position,D.quaternion,D.scale),D.projectionMatrix.fromArray(C.projectionMatrix),D.projectionMatrixInverse.copy(D.projectionMatrix).invert(),D.viewport.set(V.x,V.y,V.width,V.height),me===0&&(H.matrix.copy(D.matrix),H.matrix.decompose(H.position,H.quaternion,H.scale)),L===!0&&H.cameras.push(D)}let P=s.enabledFeatures;if(P&&P.includes("depth-sensing")&&s.depthUsage=="gpu-optimized"&&y){u=n.getBinding();let me=u.getDepthInformation(U[0]);me&&me.isValid&&me.texture&&m.init(me,s.renderState)}if(P&&P.includes("camera-access")&&y){e.state.unbindTexture(),u=n.getBinding();for(let me=0;me<U.length;me++){let C=U[me].camera;if(C){let V=g[C];V||(V=new Qr,g[C]=V);let D=u.getCameraImage(C);V.sourceTexture=D}}}}for(let U=0;U<A.length;U++){let L=w[U],P=A[U];L!==null&&P!==void 0&&P.update(L,q,l||o)}ze&&ze(pe,q),q.detectedPlanes&&n.dispatchEvent({type:"planesdetected",data:q}),p=null}let We=new hd;We.setAnimationLoop(qe),this.setAnimationLoop=function(pe){ze=pe},this.dispose=function(){}}},uy=new Je,gd=new Ze;gd.set(-1,0,0,0,1,0,0,0,1);function fy(i,e){function t(m,g){m.matrixAutoUpdate===!0&&m.updateMatrix(),g.value.copy(m.matrix)}function n(m,g){g.color.getRGB(m.fogColor.value,Ql(i)),g.isFog?(m.fogNear.value=g.near,m.fogFar.value=g.far):g.isFogExp2&&(m.fogDensity.value=g.density)}function s(m,g,E,S,x){g.isNodeMaterial?g.uniformsNeedUpdate=!1:g.isMeshBasicMaterial?r(m,g):g.isMeshLambertMaterial?(r(m,g),g.envMap&&(m.envMapIntensity.value=g.envMapIntensity)):g.isMeshToonMaterial?(r(m,g),u(m,g)):g.isMeshPhongMaterial?(r(m,g),h(m,g),g.envMap&&(m.envMapIntensity.value=g.envMapIntensity)):g.isMeshStandardMaterial?(r(m,g),f(m,g),g.isMeshPhysicalMaterial&&d(m,g,x)):g.isMeshMatcapMaterial?(r(m,g),p(m,g)):g.isMeshDepthMaterial?r(m,g):g.isMeshDistanceMaterial?(r(m,g),y(m,g)):g.isMeshNormalMaterial?r(m,g):g.isLineBasicMaterial?(o(m,g),g.isLineDashedMaterial&&a(m,g)):g.isPointsMaterial?c(m,g,E,S):g.isSpriteMaterial?l(m,g):g.isShadowMaterial?(m.color.value.copy(g.color),m.opacity.value=g.opacity):g.isShaderMaterial&&(g.uniformsNeedUpdate=!1)}function r(m,g){m.opacity.value=g.opacity,g.color&&m.diffuse.value.copy(g.color),g.emissive&&m.emissive.value.copy(g.emissive).multiplyScalar(g.emissiveIntensity),g.map&&(m.map.value=g.map,t(g.map,m.mapTransform)),g.alphaMap&&(m.alphaMap.value=g.alphaMap,t(g.alphaMap,m.alphaMapTransform)),g.bumpMap&&(m.bumpMap.value=g.bumpMap,t(g.bumpMap,m.bumpMapTransform),m.bumpScale.value=g.bumpScale,g.side===Ft&&(m.bumpScale.value*=-1)),g.normalMap&&(m.normalMap.value=g.normalMap,t(g.normalMap,m.normalMapTransform),m.normalScale.value.copy(g.normalScale),g.side===Ft&&m.normalScale.value.negate()),g.displacementMap&&(m.displacementMap.value=g.displacementMap,t(g.displacementMap,m.displacementMapTransform),m.displacementScale.value=g.displacementScale,m.displacementBias.value=g.displacementBias),g.emissiveMap&&(m.emissiveMap.value=g.emissiveMap,t(g.emissiveMap,m.emissiveMapTransform)),g.specularMap&&(m.specularMap.value=g.specularMap,t(g.specularMap,m.specularMapTransform)),g.alphaTest>0&&(m.alphaTest.value=g.alphaTest);let E=e.get(g),S=E.envMap,x=E.envMapRotation;S&&(m.envMap.value=S,m.envMapRotation.value.setFromMatrix4(uy.makeRotationFromEuler(x)).transpose(),S.isCubeTexture&&S.isRenderTargetTexture===!1&&m.envMapRotation.value.premultiply(gd),m.reflectivity.value=g.reflectivity,m.ior.value=g.ior,m.refractionRatio.value=g.refractionRatio),g.lightMap&&(m.lightMap.value=g.lightMap,m.lightMapIntensity.value=g.lightMapIntensity,t(g.lightMap,m.lightMapTransform)),g.aoMap&&(m.aoMap.value=g.aoMap,m.aoMapIntensity.value=g.aoMapIntensity,t(g.aoMap,m.aoMapTransform))}function o(m,g){m.diffuse.value.copy(g.color),m.opacity.value=g.opacity,g.map&&(m.map.value=g.map,t(g.map,m.mapTransform))}function a(m,g){m.dashSize.value=g.dashSize,m.totalSize.value=g.dashSize+g.gapSize,m.scale.value=g.scale}function c(m,g,E,S){m.diffuse.value.copy(g.color),m.opacity.value=g.opacity,m.size.value=g.size*E,m.scale.value=S*.5,g.map&&(m.map.value=g.map,t(g.map,m.uvTransform)),g.alphaMap&&(m.alphaMap.value=g.alphaMap,t(g.alphaMap,m.alphaMapTransform)),g.alphaTest>0&&(m.alphaTest.value=g.alphaTest)}function l(m,g){m.diffuse.value.copy(g.color),m.opacity.value=g.opacity,m.rotation.value=g.rotation,g.map&&(m.map.value=g.map,t(g.map,m.mapTransform)),g.alphaMap&&(m.alphaMap.value=g.alphaMap,t(g.alphaMap,m.alphaMapTransform)),g.alphaTest>0&&(m.alphaTest.value=g.alphaTest)}function h(m,g){m.specular.value.copy(g.specular),m.shininess.value=Math.max(g.shininess,1e-4)}function u(m,g){g.gradientMap&&(m.gradientMap.value=g.gradientMap)}function f(m,g){m.metalness.value=g.metalness,g.metalnessMap&&(m.metalnessMap.value=g.metalnessMap,t(g.metalnessMap,m.metalnessMapTransform)),m.roughness.value=g.roughness,g.roughnessMap&&(m.roughnessMap.value=g.roughnessMap,t(g.roughnessMap,m.roughnessMapTransform)),g.envMap&&(m.envMapIntensity.value=g.envMapIntensity)}function d(m,g,E){m.ior.value=g.ior,g.sheen>0&&(m.sheenColor.value.copy(g.sheenColor).multiplyScalar(g.sheen),m.sheenRoughness.value=g.sheenRoughness,g.sheenColorMap&&(m.sheenColorMap.value=g.sheenColorMap,t(g.sheenColorMap,m.sheenColorMapTransform)),g.sheenRoughnessMap&&(m.sheenRoughnessMap.value=g.sheenRoughnessMap,t(g.sheenRoughnessMap,m.sheenRoughnessMapTransform))),g.clearcoat>0&&(m.clearcoat.value=g.clearcoat,m.clearcoatRoughness.value=g.clearcoatRoughness,g.clearcoatMap&&(m.clearcoatMap.value=g.clearcoatMap,t(g.clearcoatMap,m.clearcoatMapTransform)),g.clearcoatRoughnessMap&&(m.clearcoatRoughnessMap.value=g.clearcoatRoughnessMap,t(g.clearcoatRoughnessMap,m.clearcoatRoughnessMapTransform)),g.clearcoatNormalMap&&(m.clearcoatNormalMap.value=g.clearcoatNormalMap,t(g.clearcoatNormalMap,m.clearcoatNormalMapTransform),m.clearcoatNormalScale.value.copy(g.clearcoatNormalScale),g.side===Ft&&m.clearcoatNormalScale.value.negate())),g.dispersion>0&&(m.dispersion.value=g.dispersion),g.iridescence>0&&(m.iridescence.value=g.iridescence,m.iridescenceIOR.value=g.iridescenceIOR,m.iridescenceThicknessMinimum.value=g.iridescenceThicknessRange[0],m.iridescenceThicknessMaximum.value=g.iridescenceThicknessRange[1],g.iridescenceMap&&(m.iridescenceMap.value=g.iridescenceMap,t(g.iridescenceMap,m.iridescenceMapTransform)),g.iridescenceThicknessMap&&(m.iridescenceThicknessMap.value=g.iridescenceThicknessMap,t(g.iridescenceThicknessMap,m.iridescenceThicknessMapTransform))),g.transmission>0&&(m.transmission.value=g.transmission,m.transmissionSamplerMap.value=E.texture,m.transmissionSamplerSize.value.set(E.width,E.height),g.transmissionMap&&(m.transmissionMap.value=g.transmissionMap,t(g.transmissionMap,m.transmissionMapTransform)),m.thickness.value=g.thickness,g.thicknessMap&&(m.thicknessMap.value=g.thicknessMap,t(g.thicknessMap,m.thicknessMapTransform)),m.attenuationDistance.value=g.attenuationDistance,m.attenuationColor.value.copy(g.attenuationColor)),g.anisotropy>0&&(m.anisotropyVector.value.set(g.anisotropy*Math.cos(g.anisotropyRotation),g.anisotropy*Math.sin(g.anisotropyRotation)),g.anisotropyMap&&(m.anisotropyMap.value=g.anisotropyMap,t(g.anisotropyMap,m.anisotropyMapTransform))),m.specularIntensity.value=g.specularIntensity,m.specularColor.value.copy(g.specularColor),g.specularColorMap&&(m.specularColorMap.value=g.specularColorMap,t(g.specularColorMap,m.specularColorMapTransform)),g.specularIntensityMap&&(m.specularIntensityMap.value=g.specularIntensityMap,t(g.specularIntensityMap,m.specularIntensityMapTransform))}function p(m,g){g.matcap&&(m.matcap.value=g.matcap)}function y(m,g){let E=e.get(g).light;m.referencePosition.value.setFromMatrixPosition(E.matrixWorld),m.nearDistance.value=E.shadow.camera.near,m.farDistance.value=E.shadow.camera.far}return{refreshFogUniforms:n,refreshMaterialUniforms:s}}function dy(i,e,t,n){let s={},r={},o=[],a=i.getParameter(i.MAX_UNIFORM_BUFFER_BINDINGS);function c(x,A){let w=A.program;n.uniformBlockBinding(x,w)}function l(x,A){let w=s[x.id];w===void 0&&(m(x),w=h(x),s[x.id]=w,x.addEventListener("dispose",E));let I=A.program;n.updateUBOMapping(x,I);let v=e.render.frame;r[x.id]!==v&&(f(x),r[x.id]=v)}function h(x){let A=u();x.__bindingPointIndex=A;let w=i.createBuffer(),I=x.__size,v=x.usage;return i.bindBuffer(i.UNIFORM_BUFFER,w),i.bufferData(i.UNIFORM_BUFFER,I,v),i.bindBuffer(i.UNIFORM_BUFFER,null),i.bindBufferBase(i.UNIFORM_BUFFER,A,w),w}function u(){for(let x=0;x<a;x++)if(o.indexOf(x)===-1)return o.push(x),x;return Ke("WebGLRenderer: Maximum number of simultaneously usable uniforms groups reached."),0}function f(x){let A=s[x.id],w=x.uniforms,I=x.__cache;i.bindBuffer(i.UNIFORM_BUFFER,A);for(let v=0,R=w.length;v<R;v++){let F=w[v];if(Array.isArray(F))for(let N=0,H=F.length;N<H;N++)d(F[N],v,N,I);else d(F,v,0,I)}i.bindBuffer(i.UNIFORM_BUFFER,null)}function d(x,A,w,I){if(y(x,A,w,I)===!0){let v=x.__offset,R=x.value;if(Array.isArray(R)){let F=0;for(let N=0;N<R.length;N++){let H=R[N],oe=g(H);p(H,x.__data,F),typeof H!="number"&&typeof H!="boolean"&&!H.isMatrix3&&!ArrayBuffer.isView(H)&&(F+=oe.storage/Float32Array.BYTES_PER_ELEMENT)}}else p(R,x.__data,0);i.bufferSubData(i.UNIFORM_BUFFER,v,x.__data)}}function p(x,A,w){typeof x=="number"||typeof x=="boolean"?A[0]=x:x.isMatrix3?(A[0]=x.elements[0],A[1]=x.elements[1],A[2]=x.elements[2],A[3]=0,A[4]=x.elements[3],A[5]=x.elements[4],A[6]=x.elements[5],A[7]=0,A[8]=x.elements[6],A[9]=x.elements[7],A[10]=x.elements[8],A[11]=0):ArrayBuffer.isView(x)?A.set(new x.constructor(x.buffer,x.byteOffset,A.length)):x.toArray(A,w)}function y(x,A,w,I){let v=x.value,R=A+"_"+w;if(I[R]===void 0)return typeof v=="number"||typeof v=="boolean"?I[R]=v:ArrayBuffer.isView(v)?I[R]=v.slice():I[R]=v.clone(),!0;{let F=I[R];if(typeof v=="number"||typeof v=="boolean"){if(F!==v)return I[R]=v,!0}else{if(ArrayBuffer.isView(v))return!0;if(F.equals(v)===!1)return F.copy(v),!0}}return!1}function m(x){let A=x.uniforms,w=0,I=16;for(let R=0,F=A.length;R<F;R++){let N=Array.isArray(A[R])?A[R]:[A[R]];for(let H=0,oe=N.length;H<oe;H++){let ie=N[H],X=Array.isArray(ie.value)?ie.value:[ie.value];for(let ee=0,Y=X.length;ee<Y;ee++){let G=X[ee],ge=g(G),ye=w%I,Me=ye%ge.boundary,Re=ye+Me;w+=Me,Re!==0&&I-Re<ge.storage&&(w+=I-Re),ie.__data=new Float32Array(ge.storage/Float32Array.BYTES_PER_ELEMENT),ie.__offset=w,w+=ge.storage}}}let v=w%I;return v>0&&(w+=I-v),x.__size=w,x.__cache={},this}function g(x){let A={boundary:0,storage:0};return typeof x=="number"||typeof x=="boolean"?(A.boundary=4,A.storage=4):x.isVector2?(A.boundary=8,A.storage=8):x.isVector3||x.isColor?(A.boundary=16,A.storage=12):x.isVector4?(A.boundary=16,A.storage=16):x.isMatrix3?(A.boundary=48,A.storage=48):x.isMatrix4?(A.boundary=64,A.storage=64):x.isTexture?ke("WebGLRenderer: Texture samplers can not be part of an uniforms group."):ArrayBuffer.isView(x)?(A.boundary=16,A.storage=x.byteLength):ke("WebGLRenderer: Unsupported uniform value type.",x),A}function E(x){let A=x.target;A.removeEventListener("dispose",E);let w=o.indexOf(A.__bindingPointIndex);o.splice(w,1),i.deleteBuffer(s[A.id]),delete s[A.id],delete r[A.id]}function S(){for(let x in s)i.deleteBuffer(s[x]);o=[],s={},r={}}return{bind:c,update:l,dispose:S}}var py=new Uint16Array([12469,15057,12620,14925,13266,14620,13807,14376,14323,13990,14545,13625,14713,13328,14840,12882,14931,12528,14996,12233,15039,11829,15066,11525,15080,11295,15085,10976,15082,10705,15073,10495,13880,14564,13898,14542,13977,14430,14158,14124,14393,13732,14556,13410,14702,12996,14814,12596,14891,12291,14937,11834,14957,11489,14958,11194,14943,10803,14921,10506,14893,10278,14858,9960,14484,14039,14487,14025,14499,13941,14524,13740,14574,13468,14654,13106,14743,12678,14818,12344,14867,11893,14889,11509,14893,11180,14881,10751,14852,10428,14812,10128,14765,9754,14712,9466,14764,13480,14764,13475,14766,13440,14766,13347,14769,13070,14786,12713,14816,12387,14844,11957,14860,11549,14868,11215,14855,10751,14825,10403,14782,10044,14729,9651,14666,9352,14599,9029,14967,12835,14966,12831,14963,12804,14954,12723,14936,12564,14917,12347,14900,11958,14886,11569,14878,11247,14859,10765,14828,10401,14784,10011,14727,9600,14660,9289,14586,8893,14508,8533,15111,12234,15110,12234,15104,12216,15092,12156,15067,12010,15028,11776,14981,11500,14942,11205,14902,10752,14861,10393,14812,9991,14752,9570,14682,9252,14603,8808,14519,8445,14431,8145,15209,11449,15208,11451,15202,11451,15190,11438,15163,11384,15117,11274,15055,10979,14994,10648,14932,10343,14871,9936,14803,9532,14729,9218,14645,8742,14556,8381,14461,8020,14365,7603,15273,10603,15272,10607,15267,10619,15256,10631,15231,10614,15182,10535,15118,10389,15042,10167,14963,9787,14883,9447,14800,9115,14710,8665,14615,8318,14514,7911,14411,7507,14279,7198,15314,9675,15313,9683,15309,9712,15298,9759,15277,9797,15229,9773,15166,9668,15084,9487,14995,9274,14898,8910,14800,8539,14697,8234,14590,7790,14479,7409,14367,7067,14178,6621,15337,8619,15337,8631,15333,8677,15325,8769,15305,8871,15264,8940,15202,8909,15119,8775,15022,8565,14916,8328,14804,8009,14688,7614,14569,7287,14448,6888,14321,6483,14088,6171,15350,7402,15350,7419,15347,7480,15340,7613,15322,7804,15287,7973,15229,8057,15148,8012,15046,7846,14933,7611,14810,7357,14682,7069,14552,6656,14421,6316,14251,5948,14007,5528,15356,5942,15356,5977,15353,6119,15348,6294,15332,6551,15302,6824,15249,7044,15171,7122,15070,7050,14949,6861,14818,6611,14679,6349,14538,6067,14398,5651,14189,5311,13935,4958,15359,4123,15359,4153,15356,4296,15353,4646,15338,5160,15311,5508,15263,5829,15188,6042,15088,6094,14966,6001,14826,5796,14678,5543,14527,5287,14377,4985,14133,4586,13869,4257,15360,1563,15360,1642,15358,2076,15354,2636,15341,3350,15317,4019,15273,4429,15203,4732,15105,4911,14981,4932,14836,4818,14679,4621,14517,4386,14359,4156,14083,3795,13808,3437,15360,122,15360,137,15358,285,15355,636,15344,1274,15322,2177,15281,2765,15215,3223,15120,3451,14995,3569,14846,3567,14681,3466,14511,3305,14344,3121,14037,2800,13753,2467,15360,0,15360,1,15359,21,15355,89,15346,253,15325,479,15287,796,15225,1148,15133,1492,15008,1749,14856,1882,14685,1886,14506,1783,14324,1608,13996,1398,13702,1183]),ii=null;function my(){return ii===null&&(ii=new Gi(py,16,16,Ji,ni),ii.name="DFG_LUT",ii.minFilter=bt,ii.magFilter=bt,ii.wrapS=Gt,ii.wrapT=Gt,ii.generateMipmaps=!1,ii.needsUpdate=!0),ii}var _r=class{constructor(e={}){let{canvas:t=If(),context:n=null,depth:s=!0,stencil:r=!1,alpha:o=!1,antialias:a=!1,premultipliedAlpha:c=!0,preserveDrawingBuffer:l=!1,powerPreference:h="default",failIfMajorPerformanceCaveat:u=!1,reversedDepthBuffer:f=!1,outputBufferType:d=hn}=e;this.isWebGLRenderer=!0;let p;if(n!==null){if(typeof WebGLRenderingContext<"u"&&n instanceof WebGLRenderingContext)throw new Error("THREE.WebGLRenderer: WebGL 1 is not supported since r163.");p=n.getContextAttributes().alpha}else p=o;let y=d,m=new Set([Za,Ya,qa]),g=new Set([hn,Wn,hr,ur,Wa,Xa]),E=new Uint32Array(4),S=new Int32Array(4),x=new B,A=null,w=null,I=[],v=[],R=null;this.domElement=t,this.debug={checkShaderErrors:!0,onShaderError:null},this.autoClear=!0,this.autoClearColor=!0,this.autoClearDepth=!0,this.autoClearStencil=!0,this.sortObjects=!0,this.clippingPlanes=[],this.localClippingEnabled=!1,this.toneMapping=Gn,this.toneMappingExposure=1,this.transmissionResolutionScale=1;let F=this,N=!1,H=null,oe=null,ie=null,X=null;this._outputColorSpace=at;let ee=0,Y=0,G=null,ge=-1,ye=null,Me=new ft,Re=new ft,ze=null,qe=new Ge(0),We=0,pe=t.width,q=t.height,U=1,L=null,P=null,z=new ft(0,0,pe,q),he=new ft(0,0,pe,q),me=!1,C=new er,V=!1,D=!1,j=new Je,$=new B,le=new ft,te={background:null,fog:null,environment:null,overrideMaterial:null,isScene:!0},ne=!1;function k(){return G===null?U:1}let b=n;function we(M,K){return t.getContext(M,K)}try{let M={alpha:!0,depth:s,stencil:r,antialias:a,premultipliedAlpha:c,preserveDrawingBuffer:l,powerPreference:h,failIfMajorPerformanceCaveat:u};if("setAttribute"in t&&t.setAttribute("data-engine",`three.js r${"185"}`),t.addEventListener("webglcontextlost",dt,!1),t.addEventListener("webglcontextrestored",ot,!1),t.addEventListener("webglcontextcreationerror",sn,!1),b===null){let K="webgl2";if(b=we(K,M),b===null)throw we(K)?new Error("THREE.WebGLRenderer: Error creating WebGL context with your selected attributes."):new Error("THREE.WebGLRenderer: Error creating WebGL context.")}}catch(M){throw Ke("WebGLRenderer: "+M.message),M}let Ae,T,_,O,W,Q,_e,Se,ae,fe,J,xe,ue,ve,Ee,De,je,Z,Pe,be,Ie,Ce,Te;function He(){Ae=new M_(b),Ae.init(),Ie=new cy(b,Ae),T=new p_(b,Ae,e,Ie),_=new oy(b,Ae),T.reversedDepthBuffer&&f&&_.buffers.depth.setReversed(!0),oe=b.createFramebuffer(),ie=b.createFramebuffer(),X=b.createFramebuffer(),O=new E_(b),W=new qx,Q=new ay(b,Ae,_,W,T,Ie,O),_e=new b_(F),Se=new Cm(b),Ce=new f_(b,Se),ae=new S_(b,Se,O,Ce),fe=new w_(b,ae,Se,Ce,O),Z=new A_(b,T,Q),Ee=new m_(W),J=new Xx(F,_e,Ae,T,Ce,Ee),xe=new fy(F,W),ue=new Zx,ve=new ey(Ae),je=new u_(F,_e,_,fe,p,c),De=new ry(F,fe,T),Te=new dy(b,O,T,_),Pe=new d_(b,Ae,O),be=new T_(b,Ae,O),O.programs=J.programs,F.capabilities=T,F.extensions=Ae,F.properties=W,F.renderLists=ue,F.shadowMap=De,F.state=_,F.info=O}He(),y!==hn&&(R=new C_(y,t.width,t.height,a,s,r));let Be=new Mh(F,b);this.xr=Be,this.getContext=function(){return b},this.getContextAttributes=function(){return b.getContextAttributes()},this.forceContextLoss=function(){let M=Ae.get("WEBGL_lose_context");M&&M.loseContext()},this.forceContextRestore=function(){let M=Ae.get("WEBGL_lose_context");M&&M.restoreContext()},this.getPixelRatio=function(){return U},this.setPixelRatio=function(M){M!==void 0&&(U=M,this.setSize(pe,q,!1))},this.getSize=function(M){return M.set(pe,q)},this.setSize=function(M,K,ce=!0){if(Be.isPresenting){ke("WebGLRenderer: Can't change size while VR device is presenting.");return}pe=M,q=K,t.width=Math.floor(M*U),t.height=Math.floor(K*U),ce===!0&&(t.style.width=M+"px",t.style.height=K+"px"),R!==null&&R.setSize(t.width,t.height),this.setViewport(0,0,M,K)},this.getDrawingBufferSize=function(M){return M.set(pe*U,q*U).floor()},this.setDrawingBufferSize=function(M,K,ce){pe=M,q=K,U=ce,t.width=Math.floor(M*ce),t.height=Math.floor(K*ce),this.setViewport(0,0,M,K)},this.setEffects=function(M){if(y===hn){Ke("WebGLRenderer: setEffects() requires outputBufferType set to HalfFloatType or FloatType.");return}if(M){for(let K=0;K<M.length;K++)if(M[K].isOutputPass===!0){ke("WebGLRenderer: OutputPass is not needed in setEffects(). Tone mapping and color space conversion are applied automatically.");break}}R.setEffects(M||[])},this.getCurrentViewport=function(M){return M.copy(Me)},this.getViewport=function(M){return M.copy(z)},this.setViewport=function(M,K,ce,se){M.isVector4?z.set(M.x,M.y,M.z,M.w):z.set(M,K,ce,se),_.viewport(Me.copy(z).multiplyScalar(U).round())},this.getScissor=function(M){return M.copy(he)},this.setScissor=function(M,K,ce,se){M.isVector4?he.set(M.x,M.y,M.z,M.w):he.set(M,K,ce,se),_.scissor(Re.copy(he).multiplyScalar(U).round())},this.getScissorTest=function(){return me},this.setScissorTest=function(M){_.setScissorTest(me=M)},this.setOpaqueSort=function(M){L=M},this.setTransparentSort=function(M){P=M},this.getClearColor=function(M){return M.copy(je.getClearColor())},this.setClearColor=function(){je.setClearColor(...arguments)},this.getClearAlpha=function(){return je.getClearAlpha()},this.setClearAlpha=function(){je.setClearAlpha(...arguments)},this.clear=function(M=!0,K=!0,ce=!0){let se=0;if(M){let re=!1;if(G!==null){let Ue=G.texture.format;re=m.has(Ue)}if(re){let Ue=G.texture.type,Fe=g.has(Ue),Ne=je.getClearColor(),Ve=je.getClearAlpha(),Xe=Ne.r,$e=Ne.g,nt=Ne.b;Fe?(E[0]=Xe,E[1]=$e,E[2]=nt,E[3]=Ve,b.clearBufferuiv(b.COLOR,0,E)):(S[0]=Xe,S[1]=$e,S[2]=nt,S[3]=Ve,b.clearBufferiv(b.COLOR,0,S))}else se|=b.COLOR_BUFFER_BIT}K&&(se|=b.DEPTH_BUFFER_BIT,this.state.buffers.depth.setMask(!0)),ce&&(se|=b.STENCIL_BUFFER_BIT,this.state.buffers.stencil.setMask(4294967295)),se!==0&&b.clear(se)},this.clearColor=function(){this.clear(!0,!1,!1)},this.clearDepth=function(){this.clear(!1,!0,!1)},this.clearStencil=function(){this.clear(!1,!1,!0)},this.setNodesHandler=function(M){M.setRenderer(this),H=M},this.dispose=function(){t.removeEventListener("webglcontextlost",dt,!1),t.removeEventListener("webglcontextrestored",ot,!1),t.removeEventListener("webglcontextcreationerror",sn,!1),je.dispose(),ue.dispose(),ve.dispose(),W.dispose(),_e.dispose(),fe.dispose(),Ce.dispose(),Te.dispose(),J.dispose(),Be.dispose(),Be.removeEventListener("sessionstart",Ei),Be.removeEventListener("sessionend",Er),ai.stop()};function dt(M){M.preventDefault(),Hr("WebGLRenderer: Context Lost."),N=!0}function ot(){Hr("WebGLRenderer: Context Restored."),N=!1;let M=O.autoReset,K=De.enabled,ce=De.autoUpdate,se=De.needsUpdate,re=De.type;He(),O.autoReset=M,De.enabled=K,De.autoUpdate=ce,De.needsUpdate=se,De.type=re}function sn(M){Ke("WebGLRenderer: A WebGL context could not be created. Reason: ",M.statusMessage)}function ct(M){let K=M.target;K.removeEventListener("dispose",ct),fn(K)}function fn(M){Cn(M),W.remove(M)}function Cn(M){let K=W.get(M).programs;K!==void 0&&(K.forEach(function(ce){J.releaseProgram(ce)}),M.isShaderMaterial&&J.releaseShaderCache(M))}this.renderBufferDirect=function(M,K,ce,se,re,Ue){K===null&&(K=te);let Fe=re.isMesh&&re.matrixWorld.determinantAffine()<0,Ne=is(M,K,ce,se,re);_.setMaterial(se,Fe);let Ve=ce.index,Xe=1;if(se.wireframe===!0){if(Ve=ae.getWireframeAttribute(ce),Ve===void 0)return;Xe=2}let $e=ce.drawRange,nt=ce.attributes.position,Ye=$e.start*Xe,pt=($e.start+$e.count)*Xe;Ue!==null&&(Ye=Math.max(Ye,Ue.start*Xe),pt=Math.min(pt,(Ue.start+Ue.count)*Xe)),Ve!==null?(Ye=Math.max(Ye,0),pt=Math.min(pt,Ve.count)):nt!=null&&(Ye=Math.max(Ye,0),pt=Math.min(pt,nt.count));let Pt=pt-Ye;if(Pt<0||Pt===1/0)return;Ce.setup(re,se,Ne,ce,Ve);let Rt,gt=Pe;if(Ve!==null&&(Rt=Se.get(Ve),gt=be,gt.setIndex(Rt)),re.isMesh)se.wireframe===!0?(_.setLineWidth(se.wireframeLinewidth*k()),gt.setMode(b.LINES)):gt.setMode(b.TRIANGLES);else if(re.isLine){let qt=se.linewidth;qt===void 0&&(qt=1),_.setLineWidth(qt*k()),re.isLineSegments?gt.setMode(b.LINES):re.isLineLoop?gt.setMode(b.LINE_LOOP):gt.setMode(b.LINE_STRIP)}else re.isPoints?gt.setMode(b.POINTS):re.isSprite&&gt.setMode(b.TRIANGLES);if(re.isBatchedMesh)if(Ae.get("WEBGL_multi_draw"))gt.renderMultiDraw(re._multiDrawStarts,re._multiDrawCounts,re._multiDrawCount);else{let qt=re._multiDrawStarts,Oe=re._multiDrawCounts,dn=re._multiDrawCount,rt=Ve?Se.get(Ve).bytesPerElement:1,bn=W.get(se).currentProgram.getUniforms();for(let Yn=0;Yn<dn;Yn++)bn.setValue(b,"_gl_DrawID",Yn),gt.render(qt[Yn]/rt,Oe[Yn])}else if(re.isInstancedMesh)gt.renderInstances(Ye,Pt,re.count);else if(ce.isInstancedBufferGeometry){let qt=ce._maxInstanceCount!==void 0?ce._maxInstanceCount:1/0,Oe=Math.min(ce.instanceCount,qt);gt.renderInstances(Ye,Pt,Oe)}else gt.render(Ye,Pt)};function Pn(M,K,ce){M.transparent===!0&&M.side===Bt&&M.forceSinglePass===!1?(M.side=Ft,M.needsUpdate=!0,ns(M,K,ce),M.side=On,M.needsUpdate=!0,ns(M,K,ce),M.side=Bt):ns(M,K,ce)}this.compile=function(M,K,ce=null){ce===null&&(ce=M),w=ve.get(ce),w.init(K),v.push(w),ce.traverseVisible(function(re){re.isLight&&re.layers.test(K.layers)&&(w.pushLight(re),re.castShadow&&w.pushShadow(re))}),M!==ce&&M.traverseVisible(function(re){re.isLight&&re.layers.test(K.layers)&&(w.pushLight(re),re.castShadow&&w.pushShadow(re))}),w.setupLights();let se=new Set;return M.traverse(function(re){if(!(re.isMesh||re.isPoints||re.isLine||re.isSprite))return;let Ue=re.material;if(Ue)if(Array.isArray(Ue))for(let Fe=0;Fe<Ue.length;Fe++){let Ne=Ue[Fe];Pn(Ne,ce,re),se.add(Ne)}else Pn(Ue,ce,re),se.add(Ue)}),w=v.pop(),se},this.compileAsync=function(M,K,ce=null){let se=this.compile(M,K,ce);return new Promise(re=>{function Ue(){if(se.forEach(function(Fe){W.get(Fe).currentProgram.isReady()&&se.delete(Fe)}),se.size===0){re(M);return}setTimeout(Ue,10)}Ae.get("KHR_parallel_shader_compile")!==null?Ue():setTimeout(Ue,10)})};let Xn=null;function oi(M){Xn&&Xn(M)}function Ei(){ai.stop()}function Er(){ai.start()}let ai=new hd;ai.setAnimationLoop(oi),typeof self<"u"&&ai.setContext(self),this.setAnimationLoop=function(M){Xn=M,Be.setAnimationLoop(M),M===null?ai.stop():ai.start()},Be.addEventListener("sessionstart",Ei),Be.addEventListener("sessionend",Er),this.render=function(M,K){if(K!==void 0&&K.isCamera!==!0){Ke("WebGLRenderer.render: camera is not an instance of THREE.Camera.");return}if(N===!0)return;H!==null&&H.renderStart(M,K);let ce=Be.enabled===!0&&Be.isPresenting===!0,se=R!==null&&(G===null||ce)&&R.begin(F,G);if(M.matrixWorldAutoUpdate===!0&&M.updateMatrixWorld(),K.parent===null&&K.matrixWorldAutoUpdate===!0&&K.updateMatrixWorld(),Be.enabled===!0&&Be.isPresenting===!0&&(R===null||R.isCompositing()===!1)&&(Be.cameraAutoUpdate===!0&&Be.updateCamera(K),K=Be.getCamera()),M.isScene===!0&&M.onBeforeRender(F,M,K,G),w=ve.get(M,v.length),w.init(K),w.state.textureUnits=Q.getTextureUnits(),v.push(w),j.multiplyMatrices(K.projectionMatrix,K.matrixWorldInverse),C.setFromProjectionMatrix(j,Un,K.reversedDepth),D=this.localClippingEnabled,V=Ee.init(this.clippingPlanes,D),A=ue.get(M,I.length),A.init(),I.push(A),Be.enabled===!0&&Be.isPresenting===!0){let Fe=F.xr.getDepthSensingMesh();Fe!==null&&Ar(Fe,K,-1/0,F.sortObjects)}Ar(M,K,0,F.sortObjects),A.finish(),F.sortObjects===!0&&A.sort(L,P,K.reversedDepth),ne=Be.enabled===!1||Be.isPresenting===!1||Be.hasDepthSensing()===!1,ne&&je.addToRenderList(A,M),this.info.render.frame++,this.info.autoReset===!0&&this.info.reset(),V===!0&&Ee.beginShadows();let re=w.state.shadowsArray;if(De.render(re,M,K),V===!0&&Ee.endShadows(),(se&&R.hasRenderPass())===!1){let Fe=A.opaque,Ne=A.transmissive;if(w.setupLights(),K.isArrayCamera){let Ve=K.cameras;if(Ne.length>0)for(let Xe=0,$e=Ve.length;Xe<$e;Xe++){let nt=Ve[Xe];ts(Fe,Ne,M,nt)}ne&&je.render(M);for(let Xe=0,$e=Ve.length;Xe<$e;Xe++){let nt=Ve[Xe];Ai(A,M,nt,nt.viewport)}}else Ne.length>0&&ts(Fe,Ne,M,K),ne&&je.render(M),Ai(A,M,K)}G!==null&&Y===0&&(Q.updateMultisampleRenderTarget(G),Q.updateRenderTargetMipmap(G)),se&&R.end(F),M.isScene===!0&&M.onAfterRender(F,M,K),Ce.resetDefaultState(),ge=-1,ye=null,v.pop(),v.length>0?(w=v[v.length-1],Q.setTextureUnits(w.state.textureUnits),V===!0&&Ee.setGlobalState(F.clippingPlanes,w.state.camera)):w=null,I.pop(),I.length>0?A=I[I.length-1]:A=null,H!==null&&H.renderEnd()};function Ar(M,K,ce,se){if(M.visible===!1)return;if(M.layers.test(K.layers)){if(M.isGroup)ce=M.renderOrder;else if(M.isLOD)M.autoUpdate===!0&&M.update(K);else if(M.isLightProbeGrid)w.pushLightProbeGrid(M);else if(M.isLight)w.pushLight(M),M.castShadow&&w.pushShadow(M);else if(M.isSprite){if(!M.frustumCulled||C.intersectsSprite(M)){se&&le.setFromMatrixPosition(M.matrixWorld).applyMatrix4(j);let Fe=fe.update(M),Ne=M.material;Ne.visible&&A.push(M,Fe,Ne,ce,le.z,null)}}else if((M.isMesh||M.isLine||M.isPoints)&&(!M.frustumCulled||C.intersectsObject(M))){let Fe=fe.update(M),Ne=M.material;if(se&&(M.boundingSphere!==void 0?(M.boundingSphere===null&&M.computeBoundingSphere(),le.copy(M.boundingSphere.center)):(Fe.boundingSphere===null&&Fe.computeBoundingSphere(),le.copy(Fe.boundingSphere.center)),le.applyMatrix4(M.matrixWorld).applyMatrix4(j)),Array.isArray(Ne)){let Ve=Fe.groups;for(let Xe=0,$e=Ve.length;Xe<$e;Xe++){let nt=Ve[Xe],Ye=Ne[nt.materialIndex];Ye&&Ye.visible&&A.push(M,Fe,Ye,ce,le.z,nt)}}else Ne.visible&&A.push(M,Fe,Ne,ce,le.z,null)}}let Ue=M.children;for(let Fe=0,Ne=Ue.length;Fe<Ne;Fe++)Ar(Ue[Fe],K,ce,se)}function Ai(M,K,ce,se){let{opaque:re,transmissive:Ue,transparent:Fe}=M;w.setupLightsView(ce),V===!0&&Ee.setGlobalState(F.clippingPlanes,ce),se&&_.viewport(Me.copy(se)),re.length>0&&wi(re,K,ce),Ue.length>0&&wi(Ue,K,ce),Fe.length>0&&wi(Fe,K,ce),_.buffers.depth.setTest(!0),_.buffers.depth.setMask(!0),_.buffers.color.setMask(!0),_.setPolygonOffset(!1)}function ts(M,K,ce,se){if((ce.isScene===!0?ce.overrideMaterial:null)!==null)return;if(w.state.transmissionRenderTarget[se.id]===void 0){let Ye=Ae.has("EXT_color_buffer_half_float")||Ae.has("EXT_color_buffer_float");w.state.transmissionRenderTarget[se.id]=new jt(1,1,{generateMipmaps:!0,type:Ye?ni:hn,minFilter:yn,samples:Math.max(4,T.samples),stencilBuffer:r,resolveDepthBuffer:!1,resolveStencilBuffer:!1,colorSpace:Qe.workingColorSpace})}let Ue=w.state.transmissionRenderTarget[se.id],Fe=se.viewport||Me;Ue.setSize(Fe.z*F.transmissionResolutionScale,Fe.w*F.transmissionResolutionScale);let Ne=F.getRenderTarget(),Ve=F.getActiveCubeFace(),Xe=F.getActiveMipmapLevel();F.setRenderTarget(Ue),F.getClearColor(qe),We=F.getClearAlpha(),We<1&&F.setClearColor(16777215,.5),F.clear(),ne&&je.render(ce);let $e=F.toneMapping;F.toneMapping=Gn;let nt=se.viewport;if(se.viewport!==void 0&&(se.viewport=void 0),w.setupLightsView(se),V===!0&&Ee.setGlobalState(F.clippingPlanes,se),wi(M,ce,se),Q.updateMultisampleRenderTarget(Ue),Q.updateRenderTargetMipmap(Ue),Ae.has("WEBGL_multisampled_render_to_texture")===!1){let Ye=!1;for(let pt=0,Pt=K.length;pt<Pt;pt++){let Rt=K[pt],{object:gt,geometry:qt,material:Oe,group:dn}=Rt;if(Oe.side===Bt&&gt.layers.test(se.layers)){let rt=Oe.side;Oe.side=Ft,Oe.needsUpdate=!0,Cs(gt,ce,se,qt,Oe,dn),Oe.side=rt,Oe.needsUpdate=!0,Ye=!0}}Ye===!0&&(Q.updateMultisampleRenderTarget(Ue),Q.updateRenderTargetMipmap(Ue))}F.setRenderTarget(Ne,Ve,Xe),F.setClearColor(qe,We),nt!==void 0&&(se.viewport=nt),F.toneMapping=$e}function wi(M,K,ce){let se=K.isScene===!0?K.overrideMaterial:null;for(let re=0,Ue=M.length;re<Ue;re++){let Fe=M[re],{object:Ne,geometry:Ve,group:Xe}=Fe,$e=Fe.material;$e.allowOverride===!0&&se!==null&&($e=se),Ne.layers.test(ce.layers)&&Cs(Ne,K,ce,Ve,$e,Xe)}}function Cs(M,K,ce,se,re,Ue){M.onBeforeRender(F,K,ce,se,re,Ue),M.modelViewMatrix.multiplyMatrices(ce.matrixWorldInverse,M.matrixWorld),M.normalMatrix.getNormalMatrix(M.modelViewMatrix),re.onBeforeRender(F,K,ce,se,M,Ue),re.transparent===!0&&re.side===Bt&&re.forceSinglePass===!1?(re.side=Ft,re.needsUpdate=!0,F.renderBufferDirect(ce,K,se,re,M,Ue),re.side=On,re.needsUpdate=!0,F.renderBufferDirect(ce,K,se,re,M,Ue),re.side=Bt):F.renderBufferDirect(ce,K,se,re,M,Ue),M.onAfterRender(F,K,ce,se,re,Ue)}function ns(M,K,ce){K.isScene!==!0&&(K=te);let se=W.get(M),re=w.state.lights,Ue=w.state.shadowsArray,Fe=re.state.version,Ne=J.getParameters(M,re.state,Ue,K,ce,w.state.lightProbeGridArray),Ve=J.getProgramCacheKey(Ne),Xe=se.programs;se.environment=M.isMeshStandardMaterial||M.isMeshLambertMaterial||M.isMeshPhongMaterial?K.environment:null,se.fog=K.fog;let $e=M.isMeshStandardMaterial||M.isMeshLambertMaterial&&!M.envMap||M.isMeshPhongMaterial&&!M.envMap;se.envMap=_e.get(M.envMap||se.environment,$e),se.envMapRotation=se.environment!==null&&M.envMap===null?K.environmentRotation:M.envMapRotation,Xe===void 0&&(M.addEventListener("dispose",ct),Xe=new Map,se.programs=Xe);let nt=Xe.get(Ve);if(nt!==void 0){if(se.currentProgram===nt&&se.lightsStateVersion===Fe)return ci(M,Ne),nt}else Ne.uniforms=J.getUniforms(M),H!==null&&M.isNodeMaterial&&H.build(M,ce,Ne),M.onBeforeCompile(Ne,F),nt=J.acquireProgram(Ne,Ve),Xe.set(Ve,nt),se.uniforms=Ne.uniforms;let Ye=se.uniforms;return(!M.isShaderMaterial&&!M.isRawShaderMaterial||M.clipping===!0)&&(Ye.clippingPlanes=Ee.uniform),ci(M,Ne),se.needsLights=Ps(M),se.lightsStateVersion=Fe,se.needsLights&&(Ye.ambientLightColor.value=re.state.ambient,Ye.lightProbe.value=re.state.probe,Ye.directionalLights.value=re.state.directional,Ye.directionalLightShadows.value=re.state.directionalShadow,Ye.spotLights.value=re.state.spot,Ye.spotLightShadows.value=re.state.spotShadow,Ye.rectAreaLights.value=re.state.rectArea,Ye.ltc_1.value=re.state.rectAreaLTC1,Ye.ltc_2.value=re.state.rectAreaLTC2,Ye.pointLights.value=re.state.point,Ye.pointLightShadows.value=re.state.pointShadow,Ye.hemisphereLights.value=re.state.hemi,Ye.directionalShadowMatrix.value=re.state.directionalShadowMatrix,Ye.spotLightMatrix.value=re.state.spotLightMatrix,Ye.spotLightMap.value=re.state.spotLightMap,Ye.pointShadowMatrix.value=re.state.pointShadowMatrix),se.lightProbeGrid=w.state.lightProbeGridArray.length>0,se.currentProgram=nt,se.uniformsList=null,nt}function Po(M){if(M.uniformsList===null){let K=M.currentProgram.getUniforms();M.uniformsList=gr.seqWithValue(K.seq,M.uniforms)}return M.uniformsList}function ci(M,K){let ce=W.get(M);ce.outputColorSpace=K.outputColorSpace,ce.batching=K.batching,ce.batchingColor=K.batchingColor,ce.instancing=K.instancing,ce.instancingColor=K.instancingColor,ce.instancingMorph=K.instancingMorph,ce.skinning=K.skinning,ce.morphTargets=K.morphTargets,ce.morphNormals=K.morphNormals,ce.morphColors=K.morphColors,ce.morphTargetsCount=K.morphTargetsCount,ce.numClippingPlanes=K.numClippingPlanes,ce.numIntersection=K.numClipIntersection,ce.vertexAlphas=K.vertexAlphas,ce.vertexTangents=K.vertexTangents,ce.toneMapping=K.toneMapping}function qn(M,K){if(M.length===0)return null;if(M.length===1)return M[0].texture!==null?M[0]:null;x.setFromMatrixPosition(K.matrixWorld);for(let ce=0,se=M.length;ce<se;ce++){let re=M[ce];if(re.texture!==null&&re.boundingBox.containsPoint(x))return re}return null}function is(M,K,ce,se,re){K.isScene!==!0&&(K=te),Q.resetTextureUnits();let Ue=K.fog,Fe=se.isMeshStandardMaterial||se.isMeshLambertMaterial||se.isMeshPhongMaterial?K.environment:null,Ne=G===null?F.outputColorSpace:G.isXRRenderTarget===!0?G.texture.colorSpace:Qe.workingColorSpace,Ve=se.isMeshStandardMaterial||se.isMeshLambertMaterial&&!se.envMap||se.isMeshPhongMaterial&&!se.envMap,Xe=_e.get(se.envMap||Fe,Ve),$e=se.vertexColors===!0&&!!ce.attributes.color&&ce.attributes.color.itemSize===4,nt=!!ce.attributes.tangent&&(!!se.normalMap||se.anisotropy>0),Ye=!!ce.morphAttributes.position,pt=!!ce.morphAttributes.normal,Pt=!!ce.morphAttributes.color,Rt=Gn;se.toneMapped&&(G===null||G.isXRRenderTarget===!0)&&(Rt=F.toneMapping);let gt=ce.morphAttributes.position||ce.morphAttributes.normal||ce.morphAttributes.color,qt=gt!==void 0?gt.length:0,Oe=W.get(se),dn=w.state.lights;if(V===!0&&(D===!0||M!==ye)){let vt=M===ye&&se.id===ge;Ee.setState(se,M,vt)}let rt=!1;se.version===Oe.__version?(Oe.needsLights&&Oe.lightsStateVersion!==dn.state.version||Oe.outputColorSpace!==Ne||re.isBatchedMesh&&Oe.batching===!1||!re.isBatchedMesh&&Oe.batching===!0||re.isBatchedMesh&&Oe.batchingColor===!0&&re.colorTexture===null||re.isBatchedMesh&&Oe.batchingColor===!1&&re.colorTexture!==null||re.isInstancedMesh&&Oe.instancing===!1||!re.isInstancedMesh&&Oe.instancing===!0||re.isSkinnedMesh&&Oe.skinning===!1||!re.isSkinnedMesh&&Oe.skinning===!0||re.isInstancedMesh&&Oe.instancingColor===!0&&re.instanceColor===null||re.isInstancedMesh&&Oe.instancingColor===!1&&re.instanceColor!==null||re.isInstancedMesh&&Oe.instancingMorph===!0&&re.morphTexture===null||re.isInstancedMesh&&Oe.instancingMorph===!1&&re.morphTexture!==null||Oe.envMap!==Xe||se.fog===!0&&Oe.fog!==Ue||Oe.numClippingPlanes!==void 0&&(Oe.numClippingPlanes!==Ee.numPlanes||Oe.numIntersection!==Ee.numIntersection)||Oe.vertexAlphas!==$e||Oe.vertexTangents!==nt||Oe.morphTargets!==Ye||Oe.morphNormals!==pt||Oe.morphColors!==Pt||Oe.toneMapping!==Rt||Oe.morphTargetsCount!==qt||!!Oe.lightProbeGrid!=w.state.lightProbeGridArray.length>0)&&(rt=!0):(rt=!0,Oe.__version=se.version);let bn=Oe.currentProgram;rt===!0&&(bn=ns(se,K,re),H&&se.isNodeMaterial&&H.onUpdateProgram(se,bn,Oe));let Yn=!1,Ci=!1,Is=!1,_t=bn.getUniforms(),It=Oe.uniforms;if(_.useProgram(bn.program)&&(Yn=!0,Ci=!0,Is=!0),se.id!==ge&&(ge=se.id,Ci=!0),Oe.needsLights){let vt=qn(w.state.lightProbeGridArray,re);Oe.lightProbeGrid!==vt&&(Oe.lightProbeGrid=vt,Ci=!0)}if(Yn||ye!==M){_.buffers.depth.getReversed()&&M.reversedDepth!==!0&&(M._reversedDepth=!0,M.updateProjectionMatrix()),_t.setValue(b,"projectionMatrix",M.projectionMatrix),_t.setValue(b,"viewMatrix",M.matrixWorldInverse);let Ii=_t.map.cameraPosition;Ii!==void 0&&Ii.setValue(b,$.setFromMatrixPosition(M.matrixWorld)),T.logarithmicDepthBuffer&&_t.setValue(b,"logDepthBufFC",2/(Math.log(M.far+1)/Math.LN2)),(se.isMeshPhongMaterial||se.isMeshToonMaterial||se.isMeshLambertMaterial||se.isMeshBasicMaterial||se.isMeshStandardMaterial||se.isShaderMaterial)&&_t.setValue(b,"isOrthographic",M.isOrthographicCamera===!0),ye!==M&&(ye=M,Ci=!0,Is=!0)}if(Oe.needsLights&&(dn.state.directionalShadowMap.length>0&&_t.setValue(b,"directionalShadowMap",dn.state.directionalShadowMap,Q),dn.state.spotShadowMap.length>0&&_t.setValue(b,"spotShadowMap",dn.state.spotShadowMap,Q),dn.state.pointShadowMap.length>0&&_t.setValue(b,"pointShadowMap",dn.state.pointShadowMap,Q)),re.isSkinnedMesh){_t.setOptional(b,re,"bindMatrix"),_t.setOptional(b,re,"bindMatrixInverse");let vt=re.skeleton;vt&&(vt.boneTexture===null&&vt.computeBoneTexture(),_t.setValue(b,"boneTexture",vt.boneTexture,Q))}re.isBatchedMesh&&(_t.setOptional(b,re,"batchingTexture"),_t.setValue(b,"batchingTexture",re._matricesTexture,Q),_t.setOptional(b,re,"batchingIdTexture"),_t.setValue(b,"batchingIdTexture",re._indirectTexture,Q),_t.setOptional(b,re,"batchingColorTexture"),re._colorsTexture!==null&&_t.setValue(b,"batchingColorTexture",re._colorsTexture,Q));let Pi=ce.morphAttributes;if((Pi.position!==void 0||Pi.normal!==void 0||Pi.color!==void 0)&&Z.update(re,ce,bn),(Ci||Oe.receiveShadow!==re.receiveShadow)&&(Oe.receiveShadow=re.receiveShadow,_t.setValue(b,"receiveShadow",re.receiveShadow)),(se.isMeshStandardMaterial||se.isMeshLambertMaterial||se.isMeshPhongMaterial)&&se.envMap===null&&K.environment!==null&&(It.envMapIntensity.value=K.environmentIntensity),It.dfgLUT!==void 0&&(It.dfgLUT.value=my()),Ci){if(_t.setValue(b,"toneMappingExposure",F.toneMappingExposure),Oe.needsLights&&Ri(It,Is),Ue&&se.fog===!0&&xe.refreshFogUniforms(It,Ue),xe.refreshMaterialUniforms(It,se,U,q,w.state.transmissionRenderTarget[M.id]),Oe.needsLights&&Oe.lightProbeGrid){let vt=Oe.lightProbeGrid;It.probesSH.value=vt.texture,It.probesMin.value.copy(vt.boundingBox.min),It.probesMax.value.copy(vt.boundingBox.max),It.probesResolution.value.copy(vt.resolution)}gr.upload(b,Po(Oe),It,Q)}if(se.isShaderMaterial&&se.uniformsNeedUpdate===!0&&(gr.upload(b,Po(Oe),It,Q),se.uniformsNeedUpdate=!1),se.isSpriteMaterial&&_t.setValue(b,"center",re.center),_t.setValue(b,"modelViewMatrix",re.modelViewMatrix),_t.setValue(b,"normalMatrix",re.normalMatrix),_t.setValue(b,"modelMatrix",re.matrixWorld),se.uniformsGroups!==void 0){let vt=se.uniformsGroups;for(let Ii=0,Ls=vt.length;Ii<Ls;Ii++){let au=vt[Ii];Te.update(au,bn),Te.bind(au,bn)}}return bn}function Ri(M,K){M.ambientLightColor.needsUpdate=K,M.lightProbe.needsUpdate=K,M.directionalLights.needsUpdate=K,M.directionalLightShadows.needsUpdate=K,M.pointLights.needsUpdate=K,M.pointLightShadows.needsUpdate=K,M.spotLights.needsUpdate=K,M.spotLightShadows.needsUpdate=K,M.rectAreaLights.needsUpdate=K,M.hemisphereLights.needsUpdate=K}function Ps(M){return M.isMeshLambertMaterial||M.isMeshToonMaterial||M.isMeshPhongMaterial||M.isMeshStandardMaterial||M.isShadowMaterial||M.isShaderMaterial&&M.lights===!0}this.getActiveCubeFace=function(){return ee},this.getActiveMipmapLevel=function(){return Y},this.getRenderTarget=function(){return G},this.setRenderTargetTextures=function(M,K,ce){let se=W.get(M);se.__autoAllocateDepthBuffer=M.resolveDepthBuffer===!1,se.__autoAllocateDepthBuffer===!1&&(se.__useRenderToTexture=!1),W.get(M.texture).__webglTexture=K,W.get(M.depthTexture).__webglTexture=se.__autoAllocateDepthBuffer?void 0:ce,se.__hasExternalTextures=!0},this.setRenderTargetFramebuffer=function(M,K){let ce=W.get(M);ce.__webglFramebuffer=K,ce.__useDefaultFramebuffer=K===void 0},this.setRenderTarget=function(M,K=0,ce=0){G=M,ee=K,Y=ce;let se=null,re=!1,Ue=!1;if(M){let Ne=W.get(M);if(Ne.__useDefaultFramebuffer!==void 0){_.bindFramebuffer(b.FRAMEBUFFER,Ne.__webglFramebuffer),Me.copy(M.viewport),Re.copy(M.scissor),ze=M.scissorTest,_.viewport(Me),_.scissor(Re),_.setScissorTest(ze),ge=-1;return}else if(Ne.__webglFramebuffer===void 0)Q.setupRenderTarget(M);else if(Ne.__hasExternalTextures)Q.rebindTextures(M,W.get(M.texture).__webglTexture,W.get(M.depthTexture).__webglTexture);else if(M.depthBuffer){let $e=M.depthTexture;if(Ne.__boundDepthTexture!==$e){if($e!==null&&W.has($e)&&(M.width!==$e.image.width||M.height!==$e.image.height))throw new Error("THREE.WebGLRenderer: Attached DepthTexture is initialized to the incorrect size.");Q.setupDepthRenderbuffer(M)}}let Ve=M.texture;(Ve.isData3DTexture||Ve.isDataArrayTexture||Ve.isCompressedArrayTexture)&&(Ue=!0);let Xe=W.get(M).__webglFramebuffer;M.isWebGLCubeRenderTarget?(Array.isArray(Xe[K])?se=Xe[K][ce]:se=Xe[K],re=!0):M.samples>0&&Q.useMultisampledRTT(M)===!1?se=W.get(M).__webglMultisampledFramebuffer:Array.isArray(Xe)?se=Xe[ce]:se=Xe,Me.copy(M.viewport),Re.copy(M.scissor),ze=M.scissorTest}else Me.copy(z).multiplyScalar(U).floor(),Re.copy(he).multiplyScalar(U).floor(),ze=me;if(ce!==0&&(se=oe),_.bindFramebuffer(b.FRAMEBUFFER,se)&&_.drawBuffers(M,se),_.viewport(Me),_.scissor(Re),_.setScissorTest(ze),re){let Ne=W.get(M.texture);b.framebufferTexture2D(b.FRAMEBUFFER,b.COLOR_ATTACHMENT0,b.TEXTURE_CUBE_MAP_POSITIVE_X+K,Ne.__webglTexture,ce)}else if(Ue){let Ne=K;for(let Ve=0;Ve<M.textures.length;Ve++){let Xe=W.get(M.textures[Ve]);b.framebufferTextureLayer(b.FRAMEBUFFER,b.COLOR_ATTACHMENT0+Ve,Xe.__webglTexture,ce,Ne)}}else if(M!==null&&ce!==0){let Ne=W.get(M.texture);b.framebufferTexture2D(b.FRAMEBUFFER,b.COLOR_ATTACHMENT0,b.TEXTURE_2D,Ne.__webglTexture,ce)}ge=-1},this.readRenderTargetPixels=function(M,K,ce,se,re,Ue,Fe,Ne=0){if(!(M&&M.isWebGLRenderTarget)){Ke("WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");return}let Ve=W.get(M).__webglFramebuffer;if(M.isWebGLCubeRenderTarget&&Fe!==void 0&&(Ve=Ve[Fe]),Ve){_.bindFramebuffer(b.FRAMEBUFFER,Ve);try{let Xe=M.textures[Ne],$e=Xe.format,nt=Xe.type;if(M.textures.length>1&&b.readBuffer(b.COLOR_ATTACHMENT0+Ne),!T.textureFormatReadable($e)){Ke("WebGLRenderer.readRenderTargetPixels: renderTarget is not in RGBA or implementation defined format.");return}if(!T.textureTypeReadable(nt)){Ke("WebGLRenderer.readRenderTargetPixels: renderTarget is not in UnsignedByteType or implementation defined type.");return}K>=0&&K<=M.width-se&&ce>=0&&ce<=M.height-re&&b.readPixels(K,ce,se,re,Ie.convert($e),Ie.convert(nt),Ue)}finally{let Xe=G!==null?W.get(G).__webglFramebuffer:null;_.bindFramebuffer(b.FRAMEBUFFER,Xe)}}},this.readRenderTargetPixelsAsync=async function(M,K,ce,se,re,Ue,Fe,Ne=0){if(!(M&&M.isWebGLRenderTarget))throw new Error("THREE.WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");let Ve=W.get(M).__webglFramebuffer;if(M.isWebGLCubeRenderTarget&&Fe!==void 0&&(Ve=Ve[Fe]),Ve)if(K>=0&&K<=M.width-se&&ce>=0&&ce<=M.height-re){_.bindFramebuffer(b.FRAMEBUFFER,Ve);let Xe=M.textures[Ne],$e=Xe.format,nt=Xe.type;if(M.textures.length>1&&b.readBuffer(b.COLOR_ATTACHMENT0+Ne),!T.textureFormatReadable($e))throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in RGBA or implementation defined format.");if(!T.textureTypeReadable(nt))throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in UnsignedByteType or implementation defined type.");let Ye=b.createBuffer();b.bindBuffer(b.PIXEL_PACK_BUFFER,Ye),b.bufferData(b.PIXEL_PACK_BUFFER,Ue.byteLength,b.STREAM_READ),b.readPixels(K,ce,se,re,Ie.convert($e),Ie.convert(nt),0);let pt=G!==null?W.get(G).__webglFramebuffer:null;_.bindFramebuffer(b.FRAMEBUFFER,pt);let Pt=b.fenceSync(b.SYNC_GPU_COMMANDS_COMPLETE,0);return b.flush(),await Df(b,Pt,4),b.bindBuffer(b.PIXEL_PACK_BUFFER,Ye),b.getBufferSubData(b.PIXEL_PACK_BUFFER,0,Ue),b.deleteBuffer(Ye),b.deleteSync(Pt),Ue}else throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: requested read bounds are out of range.")},this.copyFramebufferToTexture=function(M,K=null,ce=0){let se=Math.pow(2,-ce),re=Math.floor(M.image.width*se),Ue=Math.floor(M.image.height*se),Fe=K!==null?K.x:0,Ne=K!==null?K.y:0;Q.setTexture2D(M,0),b.copyTexSubImage2D(b.TEXTURE_2D,ce,0,0,Fe,Ne,re,Ue),_.unbindTexture()},this.copyTextureToTexture=function(M,K,ce=null,se=null,re=0,Ue=0){let Fe,Ne,Ve,Xe,$e,nt,Ye,pt,Pt,Rt=M.isCompressedTexture?M.mipmaps[Ue]:M.image;if(ce!==null)Fe=ce.max.x-ce.min.x,Ne=ce.max.y-ce.min.y,Ve=ce.isBox3?ce.max.z-ce.min.z:1,Xe=ce.min.x,$e=ce.min.y,nt=ce.isBox3?ce.min.z:0;else{let It=Math.pow(2,-re);Fe=Math.floor(Rt.width*It),Ne=Math.floor(Rt.height*It),M.isDataArrayTexture?Ve=Rt.depth:M.isData3DTexture?Ve=Math.floor(Rt.depth*It):Ve=1,Xe=0,$e=0,nt=0}se!==null?(Ye=se.x,pt=se.y,Pt=se.z):(Ye=0,pt=0,Pt=0);let gt=Ie.convert(K.format),qt=Ie.convert(K.type),Oe;K.isData3DTexture?(Q.setTexture3D(K,0),Oe=b.TEXTURE_3D):K.isDataArrayTexture||K.isCompressedArrayTexture?(Q.setTexture2DArray(K,0),Oe=b.TEXTURE_2D_ARRAY):(Q.setTexture2D(K,0),Oe=b.TEXTURE_2D),_.activeTexture(b.TEXTURE0),_.pixelStorei(b.UNPACK_FLIP_Y_WEBGL,K.flipY),_.pixelStorei(b.UNPACK_PREMULTIPLY_ALPHA_WEBGL,K.premultiplyAlpha),_.pixelStorei(b.UNPACK_ALIGNMENT,K.unpackAlignment);let dn=_.getParameter(b.UNPACK_ROW_LENGTH),rt=_.getParameter(b.UNPACK_IMAGE_HEIGHT),bn=_.getParameter(b.UNPACK_SKIP_PIXELS),Yn=_.getParameter(b.UNPACK_SKIP_ROWS),Ci=_.getParameter(b.UNPACK_SKIP_IMAGES);_.pixelStorei(b.UNPACK_ROW_LENGTH,Rt.width),_.pixelStorei(b.UNPACK_IMAGE_HEIGHT,Rt.height),_.pixelStorei(b.UNPACK_SKIP_PIXELS,Xe),_.pixelStorei(b.UNPACK_SKIP_ROWS,$e),_.pixelStorei(b.UNPACK_SKIP_IMAGES,nt);let Is=M.isDataArrayTexture||M.isData3DTexture,_t=K.isDataArrayTexture||K.isData3DTexture;if(M.isDepthTexture){let It=W.get(M),Pi=W.get(K),vt=W.get(It.__renderTarget),Ii=W.get(Pi.__renderTarget);_.bindFramebuffer(b.READ_FRAMEBUFFER,vt.__webglFramebuffer),_.bindFramebuffer(b.DRAW_FRAMEBUFFER,Ii.__webglFramebuffer);for(let Ls=0;Ls<Ve;Ls++)Is&&(b.framebufferTextureLayer(b.READ_FRAMEBUFFER,b.COLOR_ATTACHMENT0,W.get(M).__webglTexture,re,nt+Ls),b.framebufferTextureLayer(b.DRAW_FRAMEBUFFER,b.COLOR_ATTACHMENT0,W.get(K).__webglTexture,Ue,Pt+Ls)),b.blitFramebuffer(Xe,$e,Fe,Ne,Ye,pt,Fe,Ne,b.DEPTH_BUFFER_BIT,b.NEAREST);_.bindFramebuffer(b.READ_FRAMEBUFFER,null),_.bindFramebuffer(b.DRAW_FRAMEBUFFER,null)}else if(re!==0||M.isRenderTargetTexture||W.has(M)){let It=W.get(M),Pi=W.get(K);_.bindFramebuffer(b.READ_FRAMEBUFFER,ie),_.bindFramebuffer(b.DRAW_FRAMEBUFFER,X);for(let vt=0;vt<Ve;vt++)Is?b.framebufferTextureLayer(b.READ_FRAMEBUFFER,b.COLOR_ATTACHMENT0,It.__webglTexture,re,nt+vt):b.framebufferTexture2D(b.READ_FRAMEBUFFER,b.COLOR_ATTACHMENT0,b.TEXTURE_2D,It.__webglTexture,re),_t?b.framebufferTextureLayer(b.DRAW_FRAMEBUFFER,b.COLOR_ATTACHMENT0,Pi.__webglTexture,Ue,Pt+vt):b.framebufferTexture2D(b.DRAW_FRAMEBUFFER,b.COLOR_ATTACHMENT0,b.TEXTURE_2D,Pi.__webglTexture,Ue),re!==0?b.blitFramebuffer(Xe,$e,Fe,Ne,Ye,pt,Fe,Ne,b.COLOR_BUFFER_BIT,b.NEAREST):_t?b.copyTexSubImage3D(Oe,Ue,Ye,pt,Pt+vt,Xe,$e,Fe,Ne):b.copyTexSubImage2D(Oe,Ue,Ye,pt,Xe,$e,Fe,Ne);_.bindFramebuffer(b.READ_FRAMEBUFFER,null),_.bindFramebuffer(b.DRAW_FRAMEBUFFER,null)}else _t?M.isDataTexture||M.isData3DTexture?b.texSubImage3D(Oe,Ue,Ye,pt,Pt,Fe,Ne,Ve,gt,qt,Rt.data):K.isCompressedArrayTexture?b.compressedTexSubImage3D(Oe,Ue,Ye,pt,Pt,Fe,Ne,Ve,gt,Rt.data):b.texSubImage3D(Oe,Ue,Ye,pt,Pt,Fe,Ne,Ve,gt,qt,Rt):M.isDataTexture?b.texSubImage2D(b.TEXTURE_2D,Ue,Ye,pt,Fe,Ne,gt,qt,Rt.data):M.isCompressedTexture?b.compressedTexSubImage2D(b.TEXTURE_2D,Ue,Ye,pt,Rt.width,Rt.height,gt,Rt.data):b.texSubImage2D(b.TEXTURE_2D,Ue,Ye,pt,Fe,Ne,gt,qt,Rt);_.pixelStorei(b.UNPACK_ROW_LENGTH,dn),_.pixelStorei(b.UNPACK_IMAGE_HEIGHT,rt),_.pixelStorei(b.UNPACK_SKIP_PIXELS,bn),_.pixelStorei(b.UNPACK_SKIP_ROWS,Yn),_.pixelStorei(b.UNPACK_SKIP_IMAGES,Ci),Ue===0&&K.generateMipmaps&&b.generateMipmap(Oe),_.unbindTexture()},this.initRenderTarget=function(M){W.get(M).__webglFramebuffer===void 0&&Q.setupRenderTarget(M)},this.initTexture=function(M){M.isCubeTexture?Q.setTextureCube(M,0):M.isData3DTexture?Q.setTexture3D(M,0):M.isDataArrayTexture||M.isCompressedArrayTexture?Q.setTexture2DArray(M,0):Q.setTexture2D(M,0),_.unbindTexture()},this.resetState=function(){ee=0,Y=0,G=null,_.reset(),Ce.reset()},typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}get coordinateSystem(){return Un}get outputColorSpace(){return this._outputColorSpace}set outputColorSpace(e){this._outputColorSpace=e;let t=this.getContext();t.drawingBufferColorSpace=Qe._getDrawingBufferColorSpace(e),t.unpackColorSpace=Qe._getUnpackColorSpace()}};var xd={type:"change"},Th={type:"start"},vd={type:"end"},Dc=new _i,yd=new Sn,gy=Math.cos(70*Ts.DEG2RAD),Ht=new B,un=2*Math.PI,mt={NONE:-1,ROTATE:0,DOLLY:1,PAN:2,TOUCH_ROTATE:3,TOUCH_PAN:4,TOUCH_DOLLY_PAN:5,TOUCH_DOLLY_ROTATE:6},Sh=1e-6,yr=class extends po{constructor(e,t=null){super(e,t),this.state=mt.NONE,this.target=new B,this.cursor=new B,this.minDistance=0,this.maxDistance=1/0,this.minZoom=0,this.maxZoom=1/0,this.minTargetRadius=0,this.maxTargetRadius=1/0,this.minPolarAngle=0,this.maxPolarAngle=Math.PI,this.minAzimuthAngle=-1/0,this.maxAzimuthAngle=1/0,this.enableDamping=!1,this.dampingFactor=.05,this.enableZoom=!0,this.zoomSpeed=1,this.enableRotate=!0,this.rotateSpeed=1,this.keyRotateSpeed=1,this.enablePan=!0,this.panSpeed=1,this.screenSpacePanning=!0,this.keyPanSpeed=7,this.zoomToCursor=!1,this.autoRotate=!1,this.autoRotateSpeed=2,this.keys={LEFT:"ArrowLeft",UP:"ArrowUp",RIGHT:"ArrowRight",BOTTOM:"ArrowDown"},this.mouseButtons={LEFT:Yi.ROTATE,MIDDLE:Yi.DOLLY,RIGHT:Yi.PAN},this.touches={ONE:Zi.ROTATE,TWO:Zi.DOLLY_PAN},this.target0=this.target.clone(),this.position0=this.object.position.clone(),this.zoom0=this.object.zoom,this._cursorStyle="auto",this._domElementKeyEvents=null,this._lastPosition=new B,this._lastQuaternion=new Kt,this._lastTargetPosition=new B,this._quat=new Kt().setFromUnitVectors(e.up,new B(0,1,0)),this._quatInverse=this._quat.clone().invert(),this._spherical=new or,this._sphericalDelta=new or,this._scale=1,this._panOffset=new B,this._rotateStart=new de,this._rotateEnd=new de,this._rotateDelta=new de,this._panStart=new de,this._panEnd=new de,this._panDelta=new de,this._dollyStart=new de,this._dollyEnd=new de,this._dollyDelta=new de,this._dollyDirection=new B,this._mouse=new de,this._performCursorZoom=!1,this._pointers=[],this._pointerPositions={},this._controlActive=!1,this._onPointerMove=xy.bind(this),this._onPointerDown=_y.bind(this),this._onPointerUp=yy.bind(this),this._onContextMenu=Ay.bind(this),this._onMouseWheel=My.bind(this),this._onKeyDown=Sy.bind(this),this._onTouchStart=Ty.bind(this),this._onTouchMove=Ey.bind(this),this._onMouseDown=vy.bind(this),this._onMouseMove=by.bind(this),this._interceptControlDown=wy.bind(this),this._interceptControlUp=Ry.bind(this),this.domElement!==null&&this.connect(this.domElement),this.update()}set cursorStyle(e){this._cursorStyle=e,e==="grab"?this.domElement.style.cursor="grab":this.domElement.style.cursor="auto"}get cursorStyle(){return this._cursorStyle}connect(e){super.connect(e),this.domElement.addEventListener("pointerdown",this._onPointerDown),this.domElement.addEventListener("pointercancel",this._onPointerUp),this.domElement.addEventListener("contextmenu",this._onContextMenu),this.domElement.addEventListener("wheel",this._onMouseWheel,{passive:!1}),this.domElement.getRootNode().addEventListener("keydown",this._interceptControlDown,{passive:!0,capture:!0}),this.domElement.style.touchAction="none"}disconnect(){this.domElement.removeEventListener("pointerdown",this._onPointerDown),this.domElement.ownerDocument.removeEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.removeEventListener("pointerup",this._onPointerUp),this.domElement.removeEventListener("pointercancel",this._onPointerUp),this.domElement.removeEventListener("wheel",this._onMouseWheel),this.domElement.removeEventListener("contextmenu",this._onContextMenu),this.stopListenToKeyEvents(),this.domElement.getRootNode().removeEventListener("keydown",this._interceptControlDown,{capture:!0}),this.domElement.style.touchAction=""}dispose(){this.disconnect()}getPolarAngle(){return this._spherical.phi}getAzimuthalAngle(){return this._spherical.theta}getDistance(){return this.object.position.distanceTo(this.target)}listenToKeyEvents(e){e.addEventListener("keydown",this._onKeyDown),this._domElementKeyEvents=e}stopListenToKeyEvents(){this._domElementKeyEvents!==null&&(this._domElementKeyEvents.removeEventListener("keydown",this._onKeyDown),this._domElementKeyEvents=null)}saveState(){this.target0.copy(this.target),this.position0.copy(this.object.position),this.zoom0=this.object.zoom}reset(){this.target.copy(this.target0),this.object.position.copy(this.position0),this.object.zoom=this.zoom0,this.object.updateProjectionMatrix(),this.dispatchEvent(xd),this.update(),this.state=mt.NONE}pan(e,t){this._pan(e,t),this.update()}dollyIn(e){this._dollyIn(e),this.update()}dollyOut(e){this._dollyOut(e),this.update()}rotateLeft(e){this._rotateLeft(e),this.update()}rotateUp(e){this._rotateUp(e),this.update()}update(e=null){let t=this.object.position;Ht.copy(t).sub(this.target),Ht.applyQuaternion(this._quat),this._spherical.setFromVector3(Ht),this.autoRotate&&this.state===mt.NONE&&this._rotateLeft(this._getAutoRotationAngle(e)),this.enableDamping?(this._spherical.theta+=this._sphericalDelta.theta*this.dampingFactor,this._spherical.phi+=this._sphericalDelta.phi*this.dampingFactor):(this._spherical.theta+=this._sphericalDelta.theta,this._spherical.phi+=this._sphericalDelta.phi);let n=this.minAzimuthAngle,s=this.maxAzimuthAngle;isFinite(n)&&isFinite(s)&&(n<-Math.PI?n+=un:n>Math.PI&&(n-=un),s<-Math.PI?s+=un:s>Math.PI&&(s-=un),n<=s?this._spherical.theta=Math.max(n,Math.min(s,this._spherical.theta)):this._spherical.theta=this._spherical.theta>(n+s)/2?Math.max(n,this._spherical.theta):Math.min(s,this._spherical.theta)),this._spherical.phi=Math.max(this.minPolarAngle,Math.min(this.maxPolarAngle,this._spherical.phi)),this._spherical.makeSafe(),this.enableDamping===!0?this.target.addScaledVector(this._panOffset,this.dampingFactor):this.target.add(this._panOffset),this.target.sub(this.cursor),this.target.clampLength(this.minTargetRadius,this.maxTargetRadius),this.target.add(this.cursor);let r=!1;if(this.zoomToCursor&&this._performCursorZoom||this.object.isOrthographicCamera)this._spherical.radius=this._clampDistance(this._spherical.radius);else{let o=this._spherical.radius;this._spherical.radius=this._clampDistance(this._spherical.radius*this._scale),r=o!=this._spherical.radius}if(Ht.setFromSpherical(this._spherical),Ht.applyQuaternion(this._quatInverse),t.copy(this.target).add(Ht),this.object.lookAt(this.target),this.enableDamping===!0?(this._sphericalDelta.theta*=1-this.dampingFactor,this._sphericalDelta.phi*=1-this.dampingFactor,this._panOffset.multiplyScalar(1-this.dampingFactor)):(this._sphericalDelta.set(0,0,0),this._panOffset.set(0,0,0)),this.zoomToCursor&&this._performCursorZoom){let o=null;if(this.object.isPerspectiveCamera){let a=Ht.length();o=this._clampDistance(a*this._scale);let c=a-o;this.object.position.addScaledVector(this._dollyDirection,c),this.object.updateMatrixWorld(),r=!!c}else if(this.object.isOrthographicCamera){let a=new B(this._mouse.x,this._mouse.y,0);a.unproject(this.object);let c=this.object.zoom;this.object.zoom=Math.max(this.minZoom,Math.min(this.maxZoom,this.object.zoom/this._scale)),this.object.updateProjectionMatrix(),r=c!==this.object.zoom;let l=new B(this._mouse.x,this._mouse.y,0);l.unproject(this.object),this.object.position.sub(l).add(a),this.object.updateMatrixWorld(),o=Ht.length()}else console.warn("WARNING: OrbitControls.js encountered an unknown camera type - zoom to cursor disabled."),this.zoomToCursor=!1;o!==null&&(this.screenSpacePanning?this.target.set(0,0,-1).transformDirection(this.object.matrix).multiplyScalar(o).add(this.object.position):(Dc.origin.copy(this.object.position),Dc.direction.set(0,0,-1).transformDirection(this.object.matrix),Math.abs(this.object.up.dot(Dc.direction))<gy?this.object.lookAt(this.target):(yd.setFromNormalAndCoplanarPoint(this.object.up,this.target),Dc.intersectPlane(yd,this.target))))}else if(this.object.isOrthographicCamera){let o=this.object.zoom;this.object.zoom=Math.max(this.minZoom,Math.min(this.maxZoom,this.object.zoom/this._scale)),o!==this.object.zoom&&(this.object.updateProjectionMatrix(),r=!0)}return this._scale=1,this._performCursorZoom=!1,r||this._lastPosition.distanceToSquared(this.object.position)>Sh||8*(1-this._lastQuaternion.dot(this.object.quaternion))>Sh||this._lastTargetPosition.distanceToSquared(this.target)>Sh?(this.dispatchEvent(xd),this._lastPosition.copy(this.object.position),this._lastQuaternion.copy(this.object.quaternion),this._lastTargetPosition.copy(this.target),!0):!1}_getAutoRotationAngle(e){return e!==null?un/60*this.autoRotateSpeed*e:un/60/60*this.autoRotateSpeed}_getZoomScale(e){let t=Math.abs(e*.01);return Math.pow(.95,this.zoomSpeed*t)}_rotateLeft(e){this._sphericalDelta.theta-=e}_rotateUp(e){this._sphericalDelta.phi-=e}_panLeft(e,t){Ht.setFromMatrixColumn(t,0),Ht.multiplyScalar(-e),this._panOffset.add(Ht)}_panUp(e,t){this.screenSpacePanning===!0?Ht.setFromMatrixColumn(t,1):(Ht.setFromMatrixColumn(t,0),Ht.crossVectors(this.object.up,Ht)),Ht.multiplyScalar(e),this._panOffset.add(Ht)}_pan(e,t){let n=this.domElement;if(this.object.isPerspectiveCamera){let s=this.object.position;Ht.copy(s).sub(this.target);let r=Ht.length();r*=Math.tan(this.object.fov/2*Math.PI/180),this._panLeft(2*e*r/n.clientHeight,this.object.matrix),this._panUp(2*t*r/n.clientHeight,this.object.matrix)}else this.object.isOrthographicCamera?(this._panLeft(e*(this.object.right-this.object.left)/this.object.zoom/n.clientWidth,this.object.matrix),this._panUp(t*(this.object.top-this.object.bottom)/this.object.zoom/n.clientHeight,this.object.matrix)):(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - pan disabled."),this.enablePan=!1)}_dollyOut(e){this.object.isPerspectiveCamera||this.object.isOrthographicCamera?this._scale/=e:(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - dolly/zoom disabled."),this.enableZoom=!1)}_dollyIn(e){this.object.isPerspectiveCamera||this.object.isOrthographicCamera?this._scale*=e:(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - dolly/zoom disabled."),this.enableZoom=!1)}_updateZoomParameters(e,t){if(!this.zoomToCursor)return;this._performCursorZoom=!0;let n=this.domElement.getBoundingClientRect(),s=e-n.left,r=t-n.top,o=n.width,a=n.height;this._mouse.x=s/o*2-1,this._mouse.y=-(r/a)*2+1,this._dollyDirection.set(this._mouse.x,this._mouse.y,1).unproject(this.object).sub(this.object.position).normalize()}_clampDistance(e){return Math.max(this.minDistance,Math.min(this.maxDistance,e))}_handleMouseDownRotate(e){this._rotateStart.set(e.clientX,e.clientY)}_handleMouseDownDolly(e){this._updateZoomParameters(e.clientX,e.clientX),this._dollyStart.set(e.clientX,e.clientY)}_handleMouseDownPan(e){this._panStart.set(e.clientX,e.clientY)}_handleMouseMoveRotate(e){this._rotateEnd.set(e.clientX,e.clientY),this._rotateDelta.subVectors(this._rotateEnd,this._rotateStart).multiplyScalar(this.rotateSpeed);let t=this.domElement;this._rotateLeft(un*this._rotateDelta.x/t.clientHeight),this._rotateUp(un*this._rotateDelta.y/t.clientHeight),this._rotateStart.copy(this._rotateEnd),this.update()}_handleMouseMoveDolly(e){this._dollyEnd.set(e.clientX,e.clientY),this._dollyDelta.subVectors(this._dollyEnd,this._dollyStart),this._dollyDelta.y>0?this._dollyOut(this._getZoomScale(this._dollyDelta.y)):this._dollyDelta.y<0&&this._dollyIn(this._getZoomScale(this._dollyDelta.y)),this._dollyStart.copy(this._dollyEnd),this.update()}_handleMouseMovePan(e){this._panEnd.set(e.clientX,e.clientY),this._panDelta.subVectors(this._panEnd,this._panStart).multiplyScalar(this.panSpeed),this._pan(this._panDelta.x,this._panDelta.y),this._panStart.copy(this._panEnd),this.update()}_handleMouseWheel(e){this._updateZoomParameters(e.clientX,e.clientY),e.deltaY<0?this._dollyIn(this._getZoomScale(e.deltaY)):e.deltaY>0&&this._dollyOut(this._getZoomScale(e.deltaY)),this.update()}_handleKeyDown(e){let t=!1;switch(e.code){case this.keys.UP:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateUp(un*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(0,this.keyPanSpeed),t=!0;break;case this.keys.BOTTOM:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateUp(-un*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(0,-this.keyPanSpeed),t=!0;break;case this.keys.LEFT:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateLeft(un*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(this.keyPanSpeed,0),t=!0;break;case this.keys.RIGHT:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateLeft(-un*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(-this.keyPanSpeed,0),t=!0;break}t&&(e.preventDefault(),this.update())}_handleTouchStartRotate(e){if(this._pointers.length===1)this._rotateStart.set(e.pageX,e.pageY);else{let t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),s=.5*(e.pageY+t.y);this._rotateStart.set(n,s)}}_handleTouchStartPan(e){if(this._pointers.length===1)this._panStart.set(e.pageX,e.pageY);else{let t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),s=.5*(e.pageY+t.y);this._panStart.set(n,s)}}_handleTouchStartDolly(e){let t=this._getSecondPointerPosition(e),n=e.pageX-t.x,s=e.pageY-t.y,r=Math.sqrt(n*n+s*s);this._dollyStart.set(0,r)}_handleTouchStartDollyPan(e){this.enableZoom&&this._handleTouchStartDolly(e),this.enablePan&&this._handleTouchStartPan(e)}_handleTouchStartDollyRotate(e){this.enableZoom&&this._handleTouchStartDolly(e),this.enableRotate&&this._handleTouchStartRotate(e)}_handleTouchMoveRotate(e){if(this._pointers.length==1)this._rotateEnd.set(e.pageX,e.pageY);else{let n=this._getSecondPointerPosition(e),s=.5*(e.pageX+n.x),r=.5*(e.pageY+n.y);this._rotateEnd.set(s,r)}this._rotateDelta.subVectors(this._rotateEnd,this._rotateStart).multiplyScalar(this.rotateSpeed);let t=this.domElement;this._rotateLeft(un*this._rotateDelta.x/t.clientHeight),this._rotateUp(un*this._rotateDelta.y/t.clientHeight),this._rotateStart.copy(this._rotateEnd)}_handleTouchMovePan(e){if(this._pointers.length===1)this._panEnd.set(e.pageX,e.pageY);else{let t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),s=.5*(e.pageY+t.y);this._panEnd.set(n,s)}this._panDelta.subVectors(this._panEnd,this._panStart).multiplyScalar(this.panSpeed),this._pan(this._panDelta.x,this._panDelta.y),this._panStart.copy(this._panEnd)}_handleTouchMoveDolly(e){let t=this._getSecondPointerPosition(e),n=e.pageX-t.x,s=e.pageY-t.y,r=Math.sqrt(n*n+s*s);this._dollyEnd.set(0,r),this._dollyDelta.set(0,Math.pow(this._dollyEnd.y/this._dollyStart.y,this.zoomSpeed)),this._dollyOut(this._dollyDelta.y),this._dollyStart.copy(this._dollyEnd);let o=(e.pageX+t.x)*.5,a=(e.pageY+t.y)*.5;this._updateZoomParameters(o,a)}_handleTouchMoveDollyPan(e){this.enableZoom&&this._handleTouchMoveDolly(e),this.enablePan&&this._handleTouchMovePan(e)}_handleTouchMoveDollyRotate(e){this.enableZoom&&this._handleTouchMoveDolly(e),this.enableRotate&&this._handleTouchMoveRotate(e)}_addPointer(e){this._pointers.push(e.pointerId)}_removePointer(e){delete this._pointerPositions[e.pointerId];for(let t=0;t<this._pointers.length;t++)if(this._pointers[t]==e.pointerId){this._pointers.splice(t,1);return}}_isTrackingPointer(e){for(let t=0;t<this._pointers.length;t++)if(this._pointers[t]==e.pointerId)return!0;return!1}_trackPointer(e){let t=this._pointerPositions[e.pointerId];t===void 0&&(t=new de,this._pointerPositions[e.pointerId]=t),t.set(e.pageX,e.pageY)}_getSecondPointerPosition(e){let t=e.pointerId===this._pointers[0]?this._pointers[1]:this._pointers[0];return this._pointerPositions[t]}_customWheelEvent(e){let t=e.deltaMode,n={clientX:e.clientX,clientY:e.clientY,deltaY:e.deltaY};switch(t){case 1:n.deltaY*=16;break;case 2:n.deltaY*=100;break}return e.ctrlKey&&!this._controlActive&&(n.deltaY*=10),n}};function _y(i){this.enabled!==!1&&(this._pointers.length===0&&(this.domElement.setPointerCapture(i.pointerId),this.domElement.ownerDocument.addEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.addEventListener("pointerup",this._onPointerUp)),!this._isTrackingPointer(i)&&(this._addPointer(i),i.pointerType==="touch"?this._onTouchStart(i):this._onMouseDown(i),this._cursorStyle==="grab"&&(this.domElement.style.cursor="grabbing")))}function xy(i){this.enabled!==!1&&(i.pointerType==="touch"?this._onTouchMove(i):this._onMouseMove(i))}function yy(i){switch(this._removePointer(i),this._pointers.length){case 0:this.domElement.releasePointerCapture(i.pointerId),this.domElement.ownerDocument.removeEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.removeEventListener("pointerup",this._onPointerUp),this.dispatchEvent(vd),this.state=mt.NONE,this._cursorStyle==="grab"&&(this.domElement.style.cursor="grab");break;case 1:let e=this._pointers[0],t=this._pointerPositions[e];this._onTouchStart({pointerId:e,pageX:t.x,pageY:t.y});break}}function vy(i){let e;switch(i.button){case 0:e=this.mouseButtons.LEFT;break;case 1:e=this.mouseButtons.MIDDLE;break;case 2:e=this.mouseButtons.RIGHT;break;default:e=-1}switch(e){case Yi.DOLLY:if(this.enableZoom===!1)return;this._handleMouseDownDolly(i),this.state=mt.DOLLY;break;case Yi.ROTATE:if(i.ctrlKey||i.metaKey||i.shiftKey){if(this.enablePan===!1)return;this._handleMouseDownPan(i),this.state=mt.PAN}else{if(this.enableRotate===!1)return;this._handleMouseDownRotate(i),this.state=mt.ROTATE}break;case Yi.PAN:if(i.ctrlKey||i.metaKey||i.shiftKey){if(this.enableRotate===!1)return;this._handleMouseDownRotate(i),this.state=mt.ROTATE}else{if(this.enablePan===!1)return;this._handleMouseDownPan(i),this.state=mt.PAN}break;default:this.state=mt.NONE}this.state!==mt.NONE&&this.dispatchEvent(Th)}function by(i){switch(this.state){case mt.ROTATE:if(this.enableRotate===!1)return;this._handleMouseMoveRotate(i);break;case mt.DOLLY:if(this.enableZoom===!1)return;this._handleMouseMoveDolly(i);break;case mt.PAN:if(this.enablePan===!1)return;this._handleMouseMovePan(i);break}}function My(i){this.enabled===!1||this.enableZoom===!1||this.state!==mt.NONE||(i.preventDefault(),this.dispatchEvent(Th),this._handleMouseWheel(this._customWheelEvent(i)),this.dispatchEvent(vd))}function Sy(i){this.enabled!==!1&&this._handleKeyDown(i)}function Ty(i){switch(this._trackPointer(i),this._pointers.length){case 1:switch(this.touches.ONE){case Zi.ROTATE:if(this.enableRotate===!1)return;this._handleTouchStartRotate(i),this.state=mt.TOUCH_ROTATE;break;case Zi.PAN:if(this.enablePan===!1)return;this._handleTouchStartPan(i),this.state=mt.TOUCH_PAN;break;default:this.state=mt.NONE}break;case 2:switch(this.touches.TWO){case Zi.DOLLY_PAN:if(this.enableZoom===!1&&this.enablePan===!1)return;this._handleTouchStartDollyPan(i),this.state=mt.TOUCH_DOLLY_PAN;break;case Zi.DOLLY_ROTATE:if(this.enableZoom===!1&&this.enableRotate===!1)return;this._handleTouchStartDollyRotate(i),this.state=mt.TOUCH_DOLLY_ROTATE;break;default:this.state=mt.NONE}break;default:this.state=mt.NONE}this.state!==mt.NONE&&this.dispatchEvent(Th)}function Ey(i){switch(this._trackPointer(i),this.state){case mt.TOUCH_ROTATE:if(this.enableRotate===!1)return;this._handleTouchMoveRotate(i),this.update();break;case mt.TOUCH_PAN:if(this.enablePan===!1)return;this._handleTouchMovePan(i),this.update();break;case mt.TOUCH_DOLLY_PAN:if(this.enableZoom===!1&&this.enablePan===!1)return;this._handleTouchMoveDollyPan(i),this.update();break;case mt.TOUCH_DOLLY_ROTATE:if(this.enableZoom===!1&&this.enableRotate===!1)return;this._handleTouchMoveDollyRotate(i),this.update();break;default:this.state=mt.NONE}}function Ay(i){this.enabled!==!1&&i.preventDefault()}function wy(i){i.key==="Control"&&(this._controlActive=!0,this.domElement.getRootNode().addEventListener("keyup",this._interceptControlUp,{passive:!0,capture:!0}))}function Ry(i){i.key==="Control"&&(this._controlActive=!1,this.domElement.getRootNode().removeEventListener("keyup",this._interceptControlUp,{passive:!0,capture:!0}))}var Eh=new WeakMap,Cy=new URL("../libs/draco/draco_decoder.wasm",import.meta.url).toString(),Py=new URL("../libs/draco/draco_wasm_wrapper.js",import.meta.url).toString(),Iy=new URL("../libs/draco/draco_decoder.js",import.meta.url).toString(),LS={js:new URL("../libs/draco/gltf/draco_wasm_wrapper.js",import.meta.url).toString(),wasm:new URL("../libs/draco/gltf/draco_decoder.wasm",import.meta.url).toString()},vr=class extends ln{constructor(e){super(e),this.decoderPaths={js:Py,wasm:Cy,dep_js:Iy},this.decoderConfig={},this.decoderBinary=null,this.decoderPending=null,this.workerLimit=4,this.workerPool=[],this.workerNextTaskID=1,this.workerSourceURL="",this.defaultAttributeIDs={position:"POSITION",normal:"NORMAL",color:"COLOR",uv:"TEX_COORD"},this.defaultAttributeTypes={position:"Float32Array",normal:"Float32Array",color:"Float32Array",uv:"Float32Array"}}setDecoderPath(e){let{decoderPaths:t}=this;return typeof e=="object"?(t.js=e.js,t.wasm=e.wasm,t.dep_js=null):(t.js=_n.resolveURL("draco_wasm_wrapper.js",e),t.wasm=_n.resolveURL("draco_decoder.wasm",e),t.dep_js=_n.resolveURL("draco_decoder.js",e)),this}setDecoderConfig(e){return console.warn("THREE.DRACOLoader: setDecoderConfig to has been deprecated and will be removed in r194."),this.decoderConfig=e,this}setWorkerLimit(e){return this.workerLimit=e,this}load(e,t,n,s){let r=new zn(this.manager);r.setPath(this.path),r.setResponseType("arraybuffer"),r.setRequestHeader(this.requestHeader),r.setWithCredentials(this.withCredentials),r.load(e,o=>{this.parse(o,t,s)},n,s)}parse(e,t,n=()=>{}){this.decodeDracoFile(e,t,null,null,at,n).catch(n)}decodeDracoFile(e,t,n,s,r=Wt,o=()=>{}){let a={attributeIDs:n||this.defaultAttributeIDs,attributeTypes:s||this.defaultAttributeTypes,useUniqueIDs:!!n,vertexColorSpace:r};return this.decodeGeometry(e,a).then(t).catch(o)}decodeGeometry(e,t){let n=JSON.stringify(t);if(Eh.has(e)){let c=Eh.get(e);if(c.key===n)return c.promise;if(e.byteLength===0)throw new Error("THREE.DRACOLoader: Unable to re-decode a buffer with different settings. Buffer has already been transferred.")}let s,r=this.workerNextTaskID++,o=e.byteLength,a=this._getWorker(r,o).then(c=>(s=c,new Promise((l,h)=>{s._callbacks[r]={resolve:l,reject:h},s.postMessage({type:"decode",id:r,taskConfig:t,buffer:e},[e])}))).then(c=>this._createGeometry(c.geometry));return a.catch(()=>!0).then(()=>{s&&r&&this._releaseTask(s,r)}),Eh.set(e,{key:n,promise:a}),a}_createGeometry(e){let t=new wt;e.index&&t.setIndex(new yt(e.index.array,1));for(let n=0;n<e.attributes.length;n++){let{name:s,array:r,itemSize:o,stride:a,vertexColorSpace:c}=e.attributes[n],l;if(o===a)l=new yt(r,o);else{let h=new Hi(r,a);l=new Vi(h,o,0)}s==="color"&&(this._assignVertexColorSpace(l,c),l.normalized=!(r instanceof Float32Array)),t.setAttribute(s,l)}return t}_assignVertexColorSpace(e,t){if(t!==at)return;let n=new Ge;for(let s=0,r=e.count;s<r;s++)n.fromBufferAttribute(e,s),Qe.colorSpaceToWorking(n,at),e.setXYZ(s,n.r,n.g,n.b)}_loadLibrary(e,t){let n=new zn(this.manager);return n.setResponseType(t),n.setWithCredentials(this.withCredentials),new Promise((s,r)=>{n.load(e,s,void 0,r)})}preload(){return this._initDecoder(),this}_initDecoder(){if(this.decoderPending)return this.decoderPending;let e=typeof WebAssembly!="object"||this.decoderConfig.type==="js",t=[],{decoderPaths:n}=this;if(e){if(n.dep_js===null)throw new Error("THREE.DRACOLoader: WebAssembly is required when using a custom decoder paths.");t.push(this._loadLibrary(n.dep_js,"text"))}else t.push(this._loadLibrary(n.js,"text")),t.push(this._loadLibrary(n.wasm,"arraybuffer"));return this.decoderPending=Promise.all(t).then(s=>{let r=s[0];e||(this.decoderConfig.wasmBinary=s[1]);let o=Ly.toString(),a=["/* draco decoder */",r,"","/* worker */",o.substring(o.indexOf("{")+1,o.lastIndexOf("}"))].join(`
`);this.workerSourceURL=URL.createObjectURL(new Blob([a]))}),this.decoderPending}_getWorker(e,t){return this._initDecoder().then(()=>{if(this.workerPool.length<this.workerLimit){let s=new Worker(this.workerSourceURL);s._callbacks={},s._taskCosts={},s._taskLoad=0,s.postMessage({type:"init",decoderConfig:this.decoderConfig}),s.onmessage=function(r){let o=r.data;switch(o.type){case"decode":s._callbacks[o.id].resolve(o);break;case"error":s._callbacks[o.id].reject(o);break;default:console.error('THREE.DRACOLoader: Unexpected message, "'+o.type+'"')}},this.workerPool.push(s)}else this.workerPool.sort(function(s,r){return s._taskLoad>r._taskLoad?-1:1});let n=this.workerPool[this.workerPool.length-1];return n._taskCosts[e]=t,n._taskLoad+=t,n})}_releaseTask(e,t){e._taskLoad-=e._taskCosts[t],delete e._callbacks[t],delete e._taskCosts[t]}debug(){console.log("Task load: ",this.workerPool.map(e=>e._taskLoad))}dispose(){for(let e=0;e<this.workerPool.length;++e)this.workerPool[e].terminate();return this.workerPool.length=0,this.workerSourceURL!==""&&URL.revokeObjectURL(this.workerSourceURL),this}};function Ly(){let i,e;onmessage=function(o){let a=o.data;switch(a.type){case"init":i=a.decoderConfig,e=new Promise(function(h){i.onModuleLoaded=function(u){h({draco:u})},DracoDecoderModule(i)});break;case"decode":let c=a.buffer,l=a.taskConfig;e.then(h=>{let u=h.draco,f=new u.Decoder;try{let d=t(u,f,new Int8Array(c),l),p=d.attributes.map(y=>y.array.buffer);d.index&&p.push(d.index.array.buffer),self.postMessage({type:"decode",id:a.id,geometry:d},p)}catch(d){console.error(d),self.postMessage({type:"error",id:a.id,error:d.message})}finally{u.destroy(f)}});break}};function t(o,a,c,l){let h=l.attributeIDs,u=l.attributeTypes,f,d,p=a.GetEncodedGeometryType(c);if(p===o.TRIANGULAR_MESH)f=new o.Mesh,d=a.DecodeArrayToMesh(c,c.byteLength,f);else if(p===o.POINT_CLOUD)f=new o.PointCloud,d=a.DecodeArrayToPointCloud(c,c.byteLength,f);else throw new Error("THREE.DRACOLoader: Unexpected geometry type.");if(!d.ok()||f.ptr===0)throw new Error("THREE.DRACOLoader: Decoding failed: "+d.error_msg());let y={index:null,attributes:[]};for(let m in h){let g=self[u[m]],E,S;if(l.useUniqueIDs)S=h[m],E=a.GetAttributeByUniqueId(f,S);else{if(S=a.GetAttributeId(f,o[h[m]]),S===-1)continue;E=a.GetAttribute(f,S)}let x=s(o,a,f,m,g,E);m==="color"&&(x.vertexColorSpace=l.vertexColorSpace),y.attributes.push(x)}return p===o.TRIANGULAR_MESH&&(y.index=n(o,a,f)),o.destroy(f),y}function n(o,a,c){let h=c.num_faces()*3,u=h*4,f=o._malloc(u);a.GetTrianglesUInt32Array(c,u,f);let d=new Uint32Array(o.HEAPF32.buffer,f,h).slice();return o._free(f),{array:d,itemSize:1}}function s(o,a,c,l,h,u){let f=c.num_points(),d=u.num_components(),p=r(o,h),y=d*h.BYTES_PER_ELEMENT,m=Math.ceil(y/4)*4,g=m/h.BYTES_PER_ELEMENT,E=f*y,S=f*m,x=o._malloc(E);a.GetAttributeDataArrayForAllPoints(c,u,p,E,x);let A=new h(o.HEAPF32.buffer,x,E/h.BYTES_PER_ELEMENT),w;if(y===m)w=A.slice();else{w=new h(S/h.BYTES_PER_ELEMENT);let I=0;for(let v=0,R=A.length;v<R;v++){for(let F=0;F<d;F++)w[I+F]=A[v*d+F];I+=g}}return o._free(x),{name:l,count:f,itemSize:d,array:w,stride:g}}function r(o,a){switch(a){case Float32Array:return o.DT_FLOAT32;case Int8Array:return o.DT_INT8;case Int16Array:return o.DT_INT16;case Int32Array:return o.DT_INT32;case Uint8Array:return o.DT_UINT8;case Uint16Array:return o.DT_UINT16;case Uint32Array:return o.DT_UINT32}}}function Ah(i,e){if(e===jl)return console.warn("THREE.BufferGeometryUtils.toTrianglesDrawMode(): Geometry already defined as triangles."),i;if(e===dr||e===So){let t=i.getIndex();if(t===null){let o=[],a=i.getAttribute("position");if(a!==void 0){for(let c=0;c<a.count;c++)o.push(c);i.setIndex(o),t=i.getIndex()}else return console.error("THREE.BufferGeometryUtils.toTrianglesDrawMode(): Undefined position attribute. Processing not possible."),i}let n=t.count-2,s=[];if(e===dr)for(let o=1;o<=n;o++)s.push(t.getX(0)),s.push(t.getX(o)),s.push(t.getX(o+1));else for(let o=0;o<n;o++)o%2===0?(s.push(t.getX(o)),s.push(t.getX(o+1)),s.push(t.getX(o+2))):(s.push(t.getX(o+2)),s.push(t.getX(o+1)),s.push(t.getX(o)));s.length/3!==n&&console.error("THREE.BufferGeometryUtils.toTrianglesDrawMode(): Unable to generate correct amount of triangles.");let r=i.clone();return r.setIndex(s),r.clearGroups(),r}else return console.error("THREE.BufferGeometryUtils.toTrianglesDrawMode(): Unknown draw mode:",e),i}function bd(i,e=Math.PI/3){let t=i.index?i.toNonIndexed():i,n=t.attributes.position,s=n.count,r;if(n.isBufferAttribute===!0&&n.itemSize===3&&n.normalized===!1)r=n.array;else{r=new Float64Array(s*3);for(let x=0;x<s;x++)r[3*x+0]=n.getX(x),r[3*x+1]=n.getY(x),r[3*x+2]=n.getZ(x)}let o=Math.cos(e),a=(1+1e-10)*100,c=s/3,l=new Float64Array(c*3);for(let x=0;x<c;x++){let A=9*x,w=r[A+0],I=r[A+1],v=r[A+2],R=r[A+3],F=r[A+4],N=r[A+5],H=r[A+6],oe=r[A+7],ie=r[A+8],X=H-R,ee=oe-F,Y=ie-N,G=w-R,ge=I-F,ye=v-N,Me=ee*ye-Y*ge,Re=Y*G-X*ye,ze=X*ge-ee*G,qe=1/(Math.sqrt(Me*Me+Re*Re+ze*ze)||1);l[3*x+0]=Me*qe,l[3*x+1]=Re*qe,l[3*x+2]=ze*qe}let h=new Int32Array(s),u=new Int32Array(s*3),f=1;for(;f<s*2;)f<<=1;let d=f-1,p=new Int32Array(f),y=0;for(let x=0;x<s;x++){let A=3*x,w=~~(r[A+0]*a),I=~~(r[A+1]*a),v=~~(r[A+2]*a),R=(Math.imul(w,73856093)^Math.imul(I,19349663)^Math.imul(v,83492791))&d;for(;;){let F=p[R];if(F===0){let H=3*y;u[H+0]=w,u[H+1]=I,u[H+2]=v,p[R]=y+1,h[x]=y++;break}let N=3*(F-1);if(u[N+0]===w&&u[N+1]===I&&u[N+2]===v){h[x]=F-1;break}R=R+1&d}}let m=new Int32Array(y+1);for(let x=0;x<s;x++)m[h[x]+1]++;for(let x=0;x<y;x++)m[x+1]+=m[x];let g=new Int32Array(s),E=m.slice(0,y);for(let x=0;x<c;x++){let A=3*x;g[E[h[A+0]]++]=x,g[E[h[A+1]]++]=x,g[E[h[A+2]]++]=x}let S=new Float32Array(s*3);for(let x=0;x<c;x++){let A=3*x,w=l[A+0],I=l[A+1],v=l[A+2];for(let R=0;R<3;R++){let F=A+R,N=h[F],H=0,oe=0,ie=0;for(let ee=m[N],Y=m[N+1];ee<Y;ee++){let G=3*g[ee],ge=l[G+0],ye=l[G+1],Me=l[G+2];w*ge+I*ye+v*Me>o&&(H+=ge,oe+=ye,ie+=Me)}let X=1/(Math.sqrt(H*H+oe*oe+ie*ie)||1);S[3*F+0]=H*X,S[3*F+1]=oe*X,S[3*F+2]=ie*X}}return t.setAttribute("normal",new yt(S,3,!1)),t}function Md(i){let e=new Map,t=new Map,n=i.clone();return Sd(i,n,function(s,r){e.set(r,s),t.set(s,r)}),n.traverse(function(s){if(!s.isSkinnedMesh)return;let r=s,o=e.get(s),a=o.skeleton.bones;r.skeleton=o.skeleton.clone(),r.bindMatrix.copy(o.bindMatrix),r.skeleton.bones=a.map(function(c){return t.get(c)}),r.bind(r.skeleton,r.bindMatrix)}),n}function Sd(i,e,t){t(i,e);for(let n=0;n<i.children.length;n++)Sd(i.children[n],e.children[n],t)}var Mr=class extends ln{constructor(e){super(e),this.dracoLoader=null,this.ktx2Loader=null,this.meshoptDecoder=null,this.pluginCallbacks=[],this.register(function(t){return new Dh(t)}),this.register(function(t){return new Nh(t)}),this.register(function(t){return new Gh(t)}),this.register(function(t){return new Wh(t)}),this.register(function(t){return new Xh(t)}),this.register(function(t){return new Oh(t)}),this.register(function(t){return new Fh(t)}),this.register(function(t){return new Bh(t)}),this.register(function(t){return new kh(t)}),this.register(function(t){return new Lh(t)}),this.register(function(t){return new zh(t)}),this.register(function(t){return new Uh(t)}),this.register(function(t){return new Vh(t)}),this.register(function(t){return new Hh(t)}),this.register(function(t){return new Ph(t)}),this.register(function(t){return new Nc(t,it.EXT_MESHOPT_COMPRESSION)}),this.register(function(t){return new Nc(t,it.KHR_MESHOPT_COMPRESSION)}),this.register(function(t){return new qh(t)})}load(e,t,n,s){let r=this,o;if(this.resourcePath!=="")o=this.resourcePath;else if(this.path!==""){let l=_n.extractUrlBase(e);o=_n.resolveURL(l,this.path)}else o=_n.extractUrlBase(e);this.manager.itemStart(e);let a=function(l){s?s(l):console.error(l),r.manager.itemError(e),r.manager.itemEnd(e)},c=new zn(this.manager);c.setPath(this.path),c.setResponseType("arraybuffer"),c.setRequestHeader(this.requestHeader),c.setWithCredentials(this.withCredentials),c.load(e,function(l){try{r.parse(l,o,function(h){t(h),r.manager.itemEnd(e)},a)}catch(h){a(h)}},n,a)}setDRACOLoader(e){return this.dracoLoader=e,this}setKTX2Loader(e){return this.ktx2Loader=e,this}setMeshoptDecoder(e){return this.meshoptDecoder=e,this}register(e){return this.pluginCallbacks.indexOf(e)===-1&&this.pluginCallbacks.push(e),this}unregister(e){return this.pluginCallbacks.indexOf(e)!==-1&&this.pluginCallbacks.splice(this.pluginCallbacks.indexOf(e),1),this}parse(e,t,n,s){let r,o={},a={},c=new TextDecoder;if(typeof e=="string")r=JSON.parse(e);else if(e instanceof ArrayBuffer)if(c.decode(new Uint8Array(e,0,4))===Rd){try{o[it.KHR_BINARY_GLTF]=new Yh(e)}catch(u){s&&s(u);return}r=JSON.parse(o[it.KHR_BINARY_GLTF].content)}else r=JSON.parse(c.decode(e));else r=e;if(r.asset===void 0||r.asset.version[0]<2){s&&s(new Error("THREE.GLTFLoader: Unsupported asset. glTF versions >=2.0 are supported."));return}let l=new eu(r,{path:t||this.resourcePath||"",crossOrigin:this.crossOrigin,requestHeader:this.requestHeader,manager:this.manager,ktx2Loader:this.ktx2Loader,meshoptDecoder:this.meshoptDecoder});l.fileLoader.setRequestHeader(this.requestHeader);for(let h=0;h<this.pluginCallbacks.length;h++){let u=this.pluginCallbacks[h](l);u.name||console.error("THREE.GLTFLoader: Invalid plugin found: missing name"),a[u.name]=u,o[u.name]=!0}if(r.extensionsUsed)for(let h=0;h<r.extensionsUsed.length;++h){let u=r.extensionsUsed[h],f=r.extensionsRequired||[];switch(u){case it.KHR_MATERIALS_UNLIT:o[u]=new Ih;break;case it.KHR_DRACO_MESH_COMPRESSION:o[u]=new Zh(r,this.dracoLoader);break;case it.KHR_TEXTURE_TRANSFORM:o[u]=new Kh;break;case it.KHR_MESH_QUANTIZATION:o[u]=new jh;break;default:f.indexOf(u)>=0&&a[u]===void 0&&console.warn('THREE.GLTFLoader: Unknown extension "'+u+'".')}}l.setExtensions(o),l.setPlugins(a),l.parse(n,s)}parseAsync(e,t){let n=this;return new Promise(function(s,r){n.parse(e,t,s,r)})}};function Dy(){let i={};return{get:function(e){return i[e]},add:function(e,t){i[e]=t},remove:function(e){delete i[e]},removeAll:function(){i={}}}}function Lt(i,e,t){let n=i.json.materials[e];return n.extensions&&n.extensions[t]?n.extensions[t]:null}var it={KHR_BINARY_GLTF:"KHR_binary_glTF",KHR_DRACO_MESH_COMPRESSION:"KHR_draco_mesh_compression",KHR_LIGHTS_PUNCTUAL:"KHR_lights_punctual",KHR_MATERIALS_CLEARCOAT:"KHR_materials_clearcoat",KHR_MATERIALS_DISPERSION:"KHR_materials_dispersion",KHR_MATERIALS_IOR:"KHR_materials_ior",KHR_MATERIALS_SHEEN:"KHR_materials_sheen",KHR_MATERIALS_SPECULAR:"KHR_materials_specular",KHR_MATERIALS_TRANSMISSION:"KHR_materials_transmission",KHR_MATERIALS_IRIDESCENCE:"KHR_materials_iridescence",KHR_MATERIALS_ANISOTROPY:"KHR_materials_anisotropy",KHR_MATERIALS_UNLIT:"KHR_materials_unlit",KHR_MATERIALS_VOLUME:"KHR_materials_volume",KHR_TEXTURE_BASISU:"KHR_texture_basisu",KHR_TEXTURE_TRANSFORM:"KHR_texture_transform",KHR_MESH_QUANTIZATION:"KHR_mesh_quantization",KHR_MATERIALS_EMISSIVE_STRENGTH:"KHR_materials_emissive_strength",EXT_MATERIALS_BUMP:"EXT_materials_bump",EXT_TEXTURE_WEBP:"EXT_texture_webp",EXT_TEXTURE_AVIF:"EXT_texture_avif",EXT_MESHOPT_COMPRESSION:"EXT_meshopt_compression",KHR_MESHOPT_COMPRESSION:"KHR_meshopt_compression",EXT_MESH_GPU_INSTANCING:"EXT_mesh_gpu_instancing"},Ph=class{constructor(e){this.parser=e,this.name=it.KHR_LIGHTS_PUNCTUAL,this.cache={refs:{},uses:{}}}_markDefs(){let e=this.parser,t=this.parser.json.nodes||[];for(let n=0,s=t.length;n<s;n++){let r=t[n];r.extensions&&r.extensions[this.name]&&r.extensions[this.name].light!==void 0&&e._addNodeRef(this.cache,r.extensions[this.name].light)}}_loadLight(e){let t=this.parser,n="light:"+e,s=t.cache.get(n);if(s)return s;let r=t.json,c=((r.extensions&&r.extensions[this.name]||{}).lights||[])[e],l,h=new Ge(16777215);c.color!==void 0&&h.setRGB(c.color[0],c.color[1],c.color[2],Wt);let u=c.range!==void 0?c.range:0;switch(c.type){case"directional":l=new uo(h),l.target.position.set(0,0,-1),l.add(l.target);break;case"point":l=new Hn(h),l.distance=u;break;case"spot":l=new Si(h),l.distance=u,c.spot=c.spot||{},c.spot.innerConeAngle=c.spot.innerConeAngle!==void 0?c.spot.innerConeAngle:0,c.spot.outerConeAngle=c.spot.outerConeAngle!==void 0?c.spot.outerConeAngle:Math.PI/4,l.angle=c.spot.outerConeAngle,l.penumbra=1-c.spot.innerConeAngle/c.spot.outerConeAngle,l.target.position.set(0,0,-1),l.add(l.target);break;default:throw new Error("THREE.GLTFLoader: Unexpected light type: "+c.type)}return l.position.set(0,0,0),ri(l,c),c.intensity!==void 0&&(l.intensity=c.intensity),l.name=t.createUniqueName(c.name||"light_"+e),s=Promise.resolve(l),t.cache.add(n,s),s}getDependency(e,t){if(e==="light")return this._loadLight(t)}createNodeAttachment(e){let t=this,n=this.parser,r=n.json.nodes[e],a=(r.extensions&&r.extensions[this.name]||{}).light;return a===void 0?null:this._loadLight(a).then(function(c){return n._getNodeRef(t.cache,a,c)})}},Ih=class{constructor(){this.name=it.KHR_MATERIALS_UNLIT}getMaterialType(){return Ot}extendParams(e,t,n){let s=[];e.color=new Ge(1,1,1),e.opacity=1;let r=t.pbrMetallicRoughness;if(r){if(Array.isArray(r.baseColorFactor)){let o=r.baseColorFactor;e.color.setRGB(o[0],o[1],o[2],Wt),e.opacity=o[3]}r.baseColorTexture!==void 0&&s.push(n.assignTexture(e,"map",r.baseColorTexture,at))}return Promise.all(s)}},Lh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_EMISSIVE_STRENGTH}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);return n===null||n.emissiveStrength!==void 0&&(t.emissiveIntensity=n.emissiveStrength),Promise.resolve()}},Dh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_CLEARCOAT}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];if(n.clearcoatFactor!==void 0&&(t.clearcoat=n.clearcoatFactor),n.clearcoatTexture!==void 0&&s.push(this.parser.assignTexture(t,"clearcoatMap",n.clearcoatTexture)),n.clearcoatRoughnessFactor!==void 0&&(t.clearcoatRoughness=n.clearcoatRoughnessFactor),n.clearcoatRoughnessTexture!==void 0&&s.push(this.parser.assignTexture(t,"clearcoatRoughnessMap",n.clearcoatRoughnessTexture)),n.clearcoatNormalTexture!==void 0&&(s.push(this.parser.assignTexture(t,"clearcoatNormalMap",n.clearcoatNormalTexture)),n.clearcoatNormalTexture.scale!==void 0)){let r=n.clearcoatNormalTexture.scale;t.clearcoatNormalScale=new de(r,r)}return Promise.all(s)}},Nh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_DISPERSION}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);return n===null||(t.dispersion=n.dispersion!==void 0?n.dispersion:0),Promise.resolve()}},Uh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_IRIDESCENCE}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];return n.iridescenceFactor!==void 0&&(t.iridescence=n.iridescenceFactor),n.iridescenceTexture!==void 0&&s.push(this.parser.assignTexture(t,"iridescenceMap",n.iridescenceTexture)),n.iridescenceIor!==void 0&&(t.iridescenceIOR=n.iridescenceIor),t.iridescenceThicknessRange===void 0&&(t.iridescenceThicknessRange=[100,400]),n.iridescenceThicknessMinimum!==void 0&&(t.iridescenceThicknessRange[0]=n.iridescenceThicknessMinimum),n.iridescenceThicknessMaximum!==void 0&&(t.iridescenceThicknessRange[1]=n.iridescenceThicknessMaximum),n.iridescenceThicknessTexture!==void 0&&s.push(this.parser.assignTexture(t,"iridescenceThicknessMap",n.iridescenceThicknessTexture)),Promise.all(s)}},Oh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_SHEEN}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];if(t.sheenColor=new Ge(0,0,0),t.sheenRoughness=0,t.sheen=1,n.sheenColorFactor!==void 0){let r=n.sheenColorFactor;t.sheenColor.setRGB(r[0],r[1],r[2],Wt)}return n.sheenRoughnessFactor!==void 0&&(t.sheenRoughness=n.sheenRoughnessFactor),n.sheenColorTexture!==void 0&&s.push(this.parser.assignTexture(t,"sheenColorMap",n.sheenColorTexture,at)),n.sheenRoughnessTexture!==void 0&&s.push(this.parser.assignTexture(t,"sheenRoughnessMap",n.sheenRoughnessTexture)),Promise.all(s)}},Fh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_TRANSMISSION}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];return n.transmissionFactor!==void 0&&(t.transmission=n.transmissionFactor),n.transmissionTexture!==void 0&&s.push(this.parser.assignTexture(t,"transmissionMap",n.transmissionTexture)),Promise.all(s)}},Bh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_VOLUME}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];t.thickness=n.thicknessFactor!==void 0?n.thicknessFactor:0,n.thicknessTexture!==void 0&&s.push(this.parser.assignTexture(t,"thicknessMap",n.thicknessTexture)),t.attenuationDistance=n.attenuationDistance||1/0;let r=n.attenuationColor||[1,1,1];return t.attenuationColor=new Ge().setRGB(r[0],r[1],r[2],Wt),Promise.all(s)}},kh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_IOR}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);return n===null||(t.ior=n.ior!==void 0?n.ior:1.5,t.ior===0&&(t.ior=1e3)),Promise.resolve()}},zh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_SPECULAR}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];t.specularIntensity=n.specularFactor!==void 0?n.specularFactor:1,n.specularTexture!==void 0&&s.push(this.parser.assignTexture(t,"specularIntensityMap",n.specularTexture));let r=n.specularColorFactor||[1,1,1];return t.specularColor=new Ge().setRGB(r[0],r[1],r[2],Wt),n.specularColorTexture!==void 0&&s.push(this.parser.assignTexture(t,"specularColorMap",n.specularColorTexture,at)),Promise.all(s)}},Hh=class{constructor(e){this.parser=e,this.name=it.EXT_MATERIALS_BUMP}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];return t.bumpScale=n.bumpFactor!==void 0?n.bumpFactor:1,n.bumpTexture!==void 0&&s.push(this.parser.assignTexture(t,"bumpMap",n.bumpTexture)),Promise.all(s)}},Vh=class{constructor(e){this.parser=e,this.name=it.KHR_MATERIALS_ANISOTROPY}getMaterialType(e){return Lt(this.parser,e,this.name)!==null?$t:null}extendMaterialParams(e,t){let n=Lt(this.parser,e,this.name);if(n===null)return Promise.resolve();let s=[];return n.anisotropyStrength!==void 0&&(t.anisotropy=n.anisotropyStrength),n.anisotropyRotation!==void 0&&(t.anisotropyRotation=n.anisotropyRotation),n.anisotropyTexture!==void 0&&s.push(this.parser.assignTexture(t,"anisotropyMap",n.anisotropyTexture)),Promise.all(s)}},Gh=class{constructor(e){this.parser=e,this.name=it.KHR_TEXTURE_BASISU}loadTexture(e){let t=this.parser,n=t.json,s=n.textures[e];if(!s.extensions||!s.extensions[this.name])return null;let r=s.extensions[this.name],o=t.options.ktx2Loader;if(!o){if(n.extensionsRequired&&n.extensionsRequired.indexOf(this.name)>=0)throw new Error("THREE.GLTFLoader: setKTX2Loader must be called before loading KTX2 textures");return null}return t.loadTextureImage(e,r.source,o)}},Wh=class{constructor(e){this.parser=e,this.name=it.EXT_TEXTURE_WEBP}loadTexture(e){let t=this.name,n=this.parser,s=n.json,r=s.textures[e];if(!r.extensions||!r.extensions[t])return null;let o=r.extensions[t],a=s.images[o.source],c=n.textureLoader;if(a.uri){let l=n.options.manager.getHandler(a.uri);l!==null&&(c=l)}return n.loadTextureImage(e,o.source,c)}},Xh=class{constructor(e){this.parser=e,this.name=it.EXT_TEXTURE_AVIF}loadTexture(e){let t=this.name,n=this.parser,s=n.json,r=s.textures[e];if(!r.extensions||!r.extensions[t])return null;let o=r.extensions[t],a=s.images[o.source],c=n.textureLoader;if(a.uri){let l=n.options.manager.getHandler(a.uri);l!==null&&(c=l)}return n.loadTextureImage(e,o.source,c)}},Nc=class{constructor(e,t){this.name=t,this.parser=e}loadBufferView(e){let t=this.parser.json,n=t.bufferViews[e];if(n.extensions&&n.extensions[this.name]){let s=n.extensions[this.name],r=this.parser.getDependency("buffer",s.buffer),o=this.parser.options.meshoptDecoder;if(!o||!o.supported){if(t.extensionsRequired&&t.extensionsRequired.indexOf(this.name)>=0)throw new Error("THREE.GLTFLoader: setMeshoptDecoder must be called before loading compressed files");return null}return r.then(function(a){let c=s.byteOffset||0,l=s.byteLength||0,h=s.count,u=s.byteStride,f=new Uint8Array(a,c,l);return o.decodeGltfBufferAsync?o.decodeGltfBufferAsync(h,u,f,s.mode,s.filter).then(function(d){return d.buffer}):o.ready.then(function(){let d=new ArrayBuffer(h*u);return o.decodeGltfBuffer(new Uint8Array(d),h,u,f,s.mode,s.filter),d})})}else return null}},qh=class{constructor(e){this.name=it.EXT_MESH_GPU_INSTANCING,this.parser=e}createNodeMesh(e){let t=this.parser.json,n=t.nodes[e];if(!n.extensions||!n.extensions[this.name]||n.mesh===void 0)return null;let s=t.meshes[n.mesh];for(let l of s.primitives)if(l.mode!==wn.TRIANGLES&&l.mode!==wn.TRIANGLE_STRIP&&l.mode!==wn.TRIANGLE_FAN&&l.mode!==void 0)return null;let o=n.extensions[this.name].attributes,a=[],c={};for(let l in o)a.push(this.parser.getDependency("accessor",o[l]).then(h=>(c[l]=h,c[l])));return a.length<1?null:(a.push(this.parser.createNodeMesh(e)),Promise.all(a).then(l=>{let h=l.pop(),u=h.isGroup?h.children:[h],f=l[0].count,d=[];for(let p of u){let y=new Je,m=new B,g=new Kt,E=new B(1,1,1),S=new Zr(p.geometry,p.material,f);for(let x=0;x<f;x++)c.TRANSLATION&&m.fromBufferAttribute(c.TRANSLATION,x),c.ROTATION&&g.fromBufferAttribute(c.ROTATION,x),c.SCALE&&E.fromBufferAttribute(c.SCALE,x),S.setMatrixAt(x,y.compose(m,g,E));for(let x in c)if(x==="_COLOR_0"){let A=c[x];S.instanceColor=new Wi(A.array,A.itemSize,A.normalized)}else x!=="TRANSLATION"&&x!=="ROTATION"&&x!=="SCALE"&&p.geometry.setAttribute(x,c[x]);At.prototype.copy.call(S,p),this.parser.assignFinalMaterial(S),d.push(S)}return h.isGroup?(h.clear(),h.add(...d),h):d[0]}))}},Rd="glTF",wo=12,Td={JSON:1313821514,BIN:5130562},Yh=class{constructor(e){this.name=it.KHR_BINARY_GLTF,this.content=null,this.body=null;let t=new DataView(e,0,wo),n=new TextDecoder;if(this.header={magic:n.decode(new Uint8Array(e.slice(0,4))),version:t.getUint32(4,!0),length:t.getUint32(8,!0)},this.header.magic!==Rd)throw new Error("THREE.GLTFLoader: Unsupported glTF-Binary header.");if(this.header.version<2)throw new Error("THREE.GLTFLoader: Legacy binary file detected.");let s=this.header.length-wo,r=new DataView(e,wo),o=0;for(;o<s;){let a=r.getUint32(o,!0);o+=4;let c=r.getUint32(o,!0);if(o+=4,c===Td.JSON){let l=new Uint8Array(e,wo+o,a);this.content=n.decode(l)}else if(c===Td.BIN){let l=wo+o;this.body=e.slice(l,l+a)}o+=a}if(this.content===null)throw new Error("THREE.GLTFLoader: JSON content not found.")}},Zh=class{constructor(e,t){if(!t)throw new Error("THREE.GLTFLoader: No DRACOLoader instance provided.");this.name=it.KHR_DRACO_MESH_COMPRESSION,this.json=e,this.dracoLoader=t,this.dracoLoader.preload()}decodePrimitive(e,t){let n=this.json,s=this.dracoLoader,r=e.extensions[this.name].bufferView,o=e.extensions[this.name].attributes,a={},c={},l={};for(let h in o){let u=$h[h]||h.toLowerCase();a[u]=o[h]}for(let h in e.attributes){let u=$h[h]||h.toLowerCase();if(o[h]!==void 0){let f=n.accessors[e.attributes[h]],d=br[f.componentType];l[u]=d.name,c[u]=f.normalized===!0}}return t.getDependency("bufferView",r).then(function(h){return new Promise(function(u,f){s.decodeDracoFile(h,function(d){for(let p in d.attributes){let y=d.attributes[p],m=c[p];m!==void 0&&(y.normalized=m)}u(d)},a,l,Wt,f)})})}},Kh=class{constructor(){this.name=it.KHR_TEXTURE_TRANSFORM}extendTexture(e,t){return(t.texCoord===void 0||t.texCoord===e.channel)&&t.offset===void 0&&t.rotation===void 0&&t.scale===void 0||(e=e.clone(),t.texCoord!==void 0&&(e.channel=t.texCoord),t.offset!==void 0&&e.offset.fromArray(t.offset),t.rotation!==void 0&&(e.rotation=t.rotation),t.scale!==void 0&&e.repeat.fromArray(t.scale),e.needsUpdate=!0),e}},jh=class{constructor(){this.name=it.KHR_MESH_QUANTIZATION}},Uc=class extends ei{constructor(e,t,n,s){super(e,t,n,s)}copySampleValue_(e){let t=this.resultBuffer,n=this.sampleValues,s=this.valueSize,r=e*s*3+s;for(let o=0;o!==s;o++)t[o]=n[r+o];return t}interpolate_(e,t,n,s){let r=this.resultBuffer,o=this.sampleValues,a=this.valueSize,c=a*2,l=a*3,h=s-t,u=(n-t)/h,f=u*u,d=f*u,p=e*l,y=p-l,m=-2*d+3*f,g=d-f,E=1-m,S=g-f+u;for(let x=0;x!==a;x++){let A=o[y+x+a],w=o[y+x+c]*h,I=o[p+x+a],v=o[p+x]*h;r[x]=E*A+S*w+m*I+g*v}return r}},Ny=new Kt,Jh=class extends Uc{interpolate_(e,t,n,s){let r=super.interpolate_(e,t,n,s);return Ny.fromArray(r).normalize().toArray(r),r}},wn={FLOAT:5126,FLOAT_MAT3:35675,FLOAT_MAT4:35676,FLOAT_VEC2:35664,FLOAT_VEC3:35665,FLOAT_VEC4:35666,LINEAR:9729,REPEAT:10497,SAMPLER_2D:35678,POINTS:0,LINES:1,LINE_LOOP:2,LINE_STRIP:3,TRIANGLES:4,TRIANGLE_STRIP:5,TRIANGLE_FAN:6,UNSIGNED_BYTE:5121,UNSIGNED_SHORT:5123},br={5120:Int8Array,5121:Uint8Array,5122:Int16Array,5123:Uint16Array,5125:Uint32Array,5126:Float32Array},Ed={9728:St,9729:bt,9984:Va,9985:lr,9986:Ss,9987:yn},Ad={33071:Gt,33648:zi,10497:Jn},wh={SCALAR:1,VEC2:2,VEC3:3,VEC4:4,MAT2:4,MAT3:9,MAT4:16},$h={POSITION:"position",NORMAL:"normal",TANGENT:"tangent",TEXCOORD_0:"uv",TEXCOORD_1:"uv1",TEXCOORD_2:"uv2",TEXCOORD_3:"uv3",COLOR_0:"color",WEIGHTS_0:"skinWeight",JOINTS_0:"skinIndex"},es={scale:"scale",translation:"position",rotation:"quaternion",weights:"morphTargetInfluences"},Uy={CUBICSPLINE:void 0,LINEAR:ds,STEP:fs},Rh={OPAQUE:"OPAQUE",MASK:"MASK",BLEND:"BLEND"};function Oy(i){return i.DefaultMaterial===void 0&&(i.DefaultMaterial=new an({color:16777215,emissive:0,metalness:1,roughness:1,transparent:!1,depthTest:!0,side:On})),i.DefaultMaterial}function ws(i,e,t){for(let n in t.extensions)i[n]===void 0&&(e.userData.gltfExtensions=e.userData.gltfExtensions||{},e.userData.gltfExtensions[n]=t.extensions[n])}function ri(i,e){e.extras!==void 0&&(typeof e.extras=="object"?Object.assign(i.userData,e.extras):console.warn("THREE.GLTFLoader: Ignoring primitive type .extras, "+e.extras))}function Fy(i,e,t){let n=!1,s=!1,r=!1;for(let l=0,h=e.length;l<h;l++){let u=e[l];if(u.POSITION!==void 0&&(n=!0),u.NORMAL!==void 0&&(s=!0),u.COLOR_0!==void 0&&(r=!0),n&&s&&r)break}if(!n&&!s&&!r)return Promise.resolve(i);let o=[],a=[],c=[];for(let l=0,h=e.length;l<h;l++){let u=e[l];if(n){let f=u.POSITION!==void 0?t.getDependency("accessor",u.POSITION):i.attributes.position;o.push(f)}if(s){let f=u.NORMAL!==void 0?t.getDependency("accessor",u.NORMAL):i.attributes.normal;a.push(f)}if(r){let f=u.COLOR_0!==void 0?t.getDependency("accessor",u.COLOR_0):i.attributes.color;c.push(f)}}return Promise.all([Promise.all(o),Promise.all(a),Promise.all(c)]).then(function(l){let h=l[0],u=l[1],f=l[2];return n&&(i.morphAttributes.position=h),s&&(i.morphAttributes.normal=u),r&&(i.morphAttributes.color=f),i.morphTargetsRelative=!0,i})}function By(i,e){if(i.updateMorphTargets(),e.weights!==void 0)for(let t=0,n=e.weights.length;t<n;t++)i.morphTargetInfluences[t]=e.weights[t];if(e.extras&&Array.isArray(e.extras.targetNames)){let t=e.extras.targetNames;if(i.morphTargetInfluences.length===t.length){i.morphTargetDictionary={};for(let n=0,s=t.length;n<s;n++)i.morphTargetDictionary[t[n]]=n}else console.warn("THREE.GLTFLoader: Invalid extras.targetNames length. Ignoring names.")}}function ky(i){let e,t=i.extensions&&i.extensions[it.KHR_DRACO_MESH_COMPRESSION];if(t?e="draco:"+t.bufferView+":"+t.indices+":"+Ch(t.attributes):e=i.indices+":"+Ch(i.attributes)+":"+i.mode,i.targets!==void 0)for(let n=0,s=i.targets.length;n<s;n++)e+=":"+Ch(i.targets[n]);return e}function Ch(i){let e="",t=Object.keys(i).sort();for(let n=0,s=t.length;n<s;n++)e+=t[n]+":"+i[t[n]]+";";return e}function Qh(i){switch(i){case Int8Array:return 1/127;case Uint8Array:return 1/255;case Int16Array:return 1/32767;case Uint16Array:return 1/65535;default:throw new Error("THREE.GLTFLoader: Unsupported normalized accessor component type.")}}function zy(i){return i.search(/\.jpe?g($|\?)/i)>0||i.search(/^data\:image\/jpeg/)===0?"image/jpeg":i.search(/\.webp($|\?)/i)>0||i.search(/^data\:image\/webp/)===0?"image/webp":i.search(/\.ktx2($|\?)/i)>0||i.search(/^data\:image\/ktx2/)===0?"image/ktx2":"image/png"}var Hy=new Je,eu=class{constructor(e={},t={}){this.json=e,this.extensions={},this.plugins={},this.options=t,this.cache=new Dy,this.associations=new Map,this.primitiveCache={},this.nodeCache={},this.meshCache={refs:{},uses:{}},this.cameraCache={refs:{},uses:{}},this.lightCache={refs:{},uses:{}},this.sourceCache={},this.textureCache={},this.nodeNamesUsed={};let n=!1,s=-1,r=!1,o=-1;if(typeof navigator<"u"&&typeof navigator.userAgent<"u"){let a=navigator.userAgent;n=/^((?!chrome|android).)*safari/i.test(a)===!0;let c=a.match(/Version\/(\d+)/);s=n&&c?parseInt(c[1],10):-1,r=a.indexOf("Firefox")>-1,o=r?a.match(/Firefox\/([0-9]+)\./)[1]:-1}typeof createImageBitmap>"u"||n&&s<17||r&&o<98?this.textureLoader=new vs(this.options.manager):this.textureLoader=new fo(this.options.manager),this.textureLoader.setCrossOrigin(this.options.crossOrigin),this.textureLoader.setRequestHeader(this.options.requestHeader),this.fileLoader=new zn(this.options.manager),this.fileLoader.setResponseType("arraybuffer"),this.options.crossOrigin==="use-credentials"&&this.fileLoader.setWithCredentials(!0)}setExtensions(e){this.extensions=e}setPlugins(e){this.plugins=e}parse(e,t){let n=this,s=this.json,r=this.extensions;this.cache.removeAll(),this.nodeCache={},this._invokeAll(function(o){return o._markDefs&&o._markDefs()}),Promise.all(this._invokeAll(function(o){return o.beforeRoot&&o.beforeRoot()})).then(function(){return Promise.all([n.getDependencies("scene"),n.getDependencies("animation"),n.getDependencies("camera")])}).then(function(o){let a={scene:o[0][s.scene||0],scenes:o[0],animations:o[1],cameras:o[2],asset:s.asset,parser:n,userData:{}};return ws(r,a,s),ri(a,s),Promise.all(n._invokeAll(function(c){return c.afterRoot&&c.afterRoot(a)})).then(function(){for(let c of a.scenes)c.updateMatrixWorld();e(a)})}).catch(t)}_markDefs(){let e=this.json.nodes||[],t=this.json.skins||[],n=this.json.meshes||[];for(let s=0,r=t.length;s<r;s++){let o=t[s].joints;for(let a=0,c=o.length;a<c;a++)e[o[a]].isBone=!0}for(let s=0,r=e.length;s<r;s++){let o=e[s];o.mesh!==void 0&&(this._addNodeRef(this.meshCache,o.mesh),o.skin!==void 0&&(n[o.mesh].isSkinnedMesh=!0)),o.camera!==void 0&&this._addNodeRef(this.cameraCache,o.camera)}}_addNodeRef(e,t){t!==void 0&&(e.refs[t]===void 0&&(e.refs[t]=e.uses[t]=0),e.refs[t]++)}_getNodeRef(e,t,n){if(e.refs[t]<=1)return n;let s=n.clone(),r=(o,a)=>{let c=this.associations.get(o);c!=null&&this.associations.set(a,c);for(let[l,h]of o.children.entries())r(h,a.children[l])};return r(n,s),s.name+="_instance_"+e.uses[t]++,s}_invokeOne(e){let t=Object.values(this.plugins);t.push(this);for(let n=0;n<t.length;n++){let s=e(t[n]);if(s)return s}return null}_invokeAll(e){let t=Object.values(this.plugins);t.unshift(this);let n=[];for(let s=0;s<t.length;s++){let r=e(t[s]);r&&n.push(r)}return n}getDependency(e,t){let n=e+":"+t,s=this.cache.get(n);if(!s){switch(e){case"scene":s=this.loadScene(t);break;case"node":s=this._invokeOne(function(r){return r.loadNode&&r.loadNode(t)});break;case"mesh":s=this._invokeOne(function(r){return r.loadMesh&&r.loadMesh(t)});break;case"accessor":s=this.loadAccessor(t);break;case"bufferView":s=this._invokeOne(function(r){return r.loadBufferView&&r.loadBufferView(t)});break;case"buffer":s=this.loadBuffer(t);break;case"material":s=this._invokeOne(function(r){return r.loadMaterial&&r.loadMaterial(t)});break;case"texture":s=this._invokeOne(function(r){return r.loadTexture&&r.loadTexture(t)});break;case"skin":s=this.loadSkin(t);break;case"animation":s=this._invokeOne(function(r){return r.loadAnimation&&r.loadAnimation(t)});break;case"camera":s=this.loadCamera(t);break;default:if(s=this._invokeOne(function(r){return r!=this&&r.getDependency&&r.getDependency(e,t)}),!s)throw new Error("Unknown type: "+e);break}this.cache.add(n,s)}return s}getDependencies(e){let t=this.cache.get(e);if(!t){let n=this,s=this.json[e+(e==="mesh"?"es":"s")]||[];t=Promise.all(s.map(function(r,o){return n.getDependency(e,o)})),this.cache.add(e,t)}return t}loadBuffer(e){let t=this.json.buffers[e],n=this.fileLoader;if(t.type&&t.type!=="arraybuffer")throw new Error("THREE.GLTFLoader: "+t.type+" buffer type is not supported.");if(t.uri===void 0&&e===0)return Promise.resolve(this.extensions[it.KHR_BINARY_GLTF].body);let s=this.options;return new Promise(function(r,o){n.load(_n.resolveURL(t.uri,s.path),r,void 0,function(){o(new Error('THREE.GLTFLoader: Failed to load buffer "'+t.uri+'".'))})})}loadBufferView(e){let t=this.json.bufferViews[e];return this.getDependency("buffer",t.buffer).then(function(n){let s=t.byteLength||0,r=t.byteOffset||0;return n.slice(r,r+s)})}loadAccessor(e){let t=this,n=this.json,s=this.json.accessors[e];if(s.bufferView===void 0&&s.sparse===void 0){let o=wh[s.type],a=br[s.componentType],c=s.normalized===!0,l=new a(s.count*o);return Promise.resolve(new yt(l,o,c))}let r=[];return s.bufferView!==void 0?r.push(this.getDependency("bufferView",s.bufferView)):r.push(null),s.sparse!==void 0&&(r.push(this.getDependency("bufferView",s.sparse.indices.bufferView)),r.push(this.getDependency("bufferView",s.sparse.values.bufferView))),Promise.all(r).then(function(o){let a=o[0],c=wh[s.type],l=br[s.componentType],h=l.BYTES_PER_ELEMENT,u=h*c,f=s.byteOffset||0,d=s.bufferView!==void 0?n.bufferViews[s.bufferView].byteStride:void 0,p=s.normalized===!0,y,m;if(d&&d!==u){let g=Math.floor(f/d),E="InterleavedBuffer:"+s.bufferView+":"+s.componentType+":"+g+":"+s.count,S=t.cache.get(E);S||(y=new l(a,g*d,s.count*d/h),S=new Hi(y,d/h),t.cache.add(E,S)),m=new Vi(S,c,f%d/h,p)}else a===null?y=new l(s.count*c):y=new l(a,f,s.count*c),m=new yt(y,c,p);if(s.sparse!==void 0){let g=wh.SCALAR,E=br[s.sparse.indices.componentType],S=s.sparse.indices.byteOffset||0,x=s.sparse.values.byteOffset||0,A=new E(o[1],S,s.sparse.count*g),w=new l(o[2],x,s.sparse.count*c);a!==null&&(m=new yt(m.array.slice(),m.itemSize,m.normalized)),m.normalized=!1;for(let I=0,v=A.length;I<v;I++){let R=A[I];if(m.setX(R,w[I*c]),c>=2&&m.setY(R,w[I*c+1]),c>=3&&m.setZ(R,w[I*c+2]),c>=4&&m.setW(R,w[I*c+3]),c>=5)throw new Error("THREE.GLTFLoader: Unsupported itemSize in sparse BufferAttribute.")}m.normalized=p}return m})}loadTexture(e){let t=this.json,n=this.options,r=t.textures[e].source,o=t.images[r],a=this.textureLoader;if(o.uri){let c=n.manager.getHandler(o.uri);c!==null&&(a=c)}return this.loadTextureImage(e,r,a)}loadTextureImage(e,t,n){let s=this,r=this.json,o=r.textures[e],a=r.images[t],c=(a.uri||a.bufferView)+":"+o.sampler;if(this.textureCache[c])return this.textureCache[c];let l=this.loadImageSource(t,n).then(function(h){h.flipY=!1,h.name=o.name||a.name||"",h.name===""&&typeof a.uri=="string"&&a.uri.startsWith("data:image/")===!1&&(h.name=a.uri);let f=(r.samplers||{})[o.sampler]||{};return h.magFilter=Ed[f.magFilter]||bt,h.minFilter=Ed[f.minFilter]||yn,h.wrapS=Ad[f.wrapS]||Jn,h.wrapT=Ad[f.wrapT]||Jn,h.generateMipmaps=!h.isCompressedTexture&&h.minFilter!==St&&h.minFilter!==bt,s.associations.set(h,{textures:e}),h}).catch(function(){return null});return this.textureCache[c]=l,l}loadImageSource(e,t){let n=this,s=this.json,r=this.options;if(this.sourceCache[e]!==void 0)return this.sourceCache[e].then(u=>u.clone());let o=s.images[e],a=self.URL||self.webkitURL,c=o.uri||"",l=!1;if(o.bufferView!==void 0)c=n.getDependency("bufferView",o.bufferView).then(function(u){l=!0;let f=new Blob([u],{type:o.mimeType});return c=a.createObjectURL(f),c});else if(o.uri===void 0)throw new Error("THREE.GLTFLoader: Image "+e+" is missing URI and bufferView");let h=Promise.resolve(c).then(function(u){return new Promise(function(f,d){let p=f;t.isImageBitmapLoader===!0&&(p=function(y){let m=new Tt(y);m.needsUpdate=!0,f(m)}),t.load(_n.resolveURL(u,r.path),p,void 0,d)})}).then(function(u){return l===!0&&a.revokeObjectURL(c),ri(u,o),u.userData.mimeType=o.mimeType||zy(o.uri),u}).catch(function(u){throw console.error("THREE.GLTFLoader: Couldn't load texture",c),u});return this.sourceCache[e]=h,h}assignTexture(e,t,n,s){let r=this;return this.getDependency("texture",n.index).then(function(o){if(!o)return null;if(n.texCoord!==void 0&&n.texCoord>0&&(o=o.clone(),o.channel=n.texCoord),r.extensions[it.KHR_TEXTURE_TRANSFORM]){let a=n.extensions!==void 0?n.extensions[it.KHR_TEXTURE_TRANSFORM]:void 0;if(a){let c=r.associations.get(o);o=r.extensions[it.KHR_TEXTURE_TRANSFORM].extendTexture(o,a),r.associations.set(o,c)}}return s!==void 0&&(o.colorSpace=s),e[t]=o,o})}assignFinalMaterial(e){let t=e.geometry,n=e.material,s=t.attributes.tangent===void 0,r=t.attributes.color!==void 0,o=t.attributes.normal===void 0;if(e.isPoints){let a="PointsMaterial:"+n.uuid,c=this.cache.get(a);c||(c=new nr,on.prototype.copy.call(c,n),c.color.copy(n.color),c.map=n.map,c.sizeAttenuation=!1,this.cache.add(a,c)),n=c}else if(e.isLine){let a="LineBasicMaterial:"+n.uuid,c=this.cache.get(a);c||(c=new tr,on.prototype.copy.call(c,n),c.color.copy(n.color),c.map=n.map,this.cache.add(a,c)),n=c}if(s||r||o){let a="ClonedMaterial:"+n.uuid+":";s&&(a+="derivative-tangents:"),r&&(a+="vertex-colors:"),o&&(a+="flat-shading:");let c=this.cache.get(a);c||(c=n.clone(),r&&(c.vertexColors=!0),o&&(c.flatShading=!0),s&&(c.normalScale&&(c.normalScale.y*=-1),c.clearcoatNormalScale&&(c.clearcoatNormalScale.y*=-1)),this.cache.add(a,c),this.associations.set(c,this.associations.get(n))),n=c}e.material=n}getMaterialType(){return an}loadMaterial(e){let t=this,n=this.json,s=this.extensions,r=n.materials[e],o,a={},c=r.extensions||{},l=[];if(c[it.KHR_MATERIALS_UNLIT]){let u=s[it.KHR_MATERIALS_UNLIT];o=u.getMaterialType(),l.push(u.extendParams(a,r,t))}else{let u=r.pbrMetallicRoughness||{};if(a.color=new Ge(1,1,1),a.opacity=1,Array.isArray(u.baseColorFactor)){let f=u.baseColorFactor;a.color.setRGB(f[0],f[1],f[2],Wt),a.opacity=f[3]}u.baseColorTexture!==void 0&&l.push(t.assignTexture(a,"map",u.baseColorTexture,at)),a.metalness=u.metallicFactor!==void 0?u.metallicFactor:1,a.roughness=u.roughnessFactor!==void 0?u.roughnessFactor:1,u.metallicRoughnessTexture!==void 0&&(l.push(t.assignTexture(a,"metalnessMap",u.metallicRoughnessTexture)),l.push(t.assignTexture(a,"roughnessMap",u.metallicRoughnessTexture))),o=this._invokeOne(function(f){return f.getMaterialType&&f.getMaterialType(e)}),l.push(Promise.all(this._invokeAll(function(f){return f.extendMaterialParams&&f.extendMaterialParams(e,a)})))}r.doubleSided===!0&&(a.side=Bt);let h=r.alphaMode||Rh.OPAQUE;if(h===Rh.BLEND?(a.transparent=!0,a.depthWrite=!1):(a.transparent=!1,h===Rh.MASK&&(a.alphaTest=r.alphaCutoff!==void 0?r.alphaCutoff:.5)),r.normalTexture!==void 0&&o!==Ot&&(l.push(t.assignTexture(a,"normalMap",r.normalTexture)),a.normalScale=new de(1,1),r.normalTexture.scale!==void 0)){let u=r.normalTexture.scale;a.normalScale.set(u,u)}if(r.occlusionTexture!==void 0&&o!==Ot&&(l.push(t.assignTexture(a,"aoMap",r.occlusionTexture)),r.occlusionTexture.strength!==void 0&&(a.aoMapIntensity=r.occlusionTexture.strength)),r.emissiveFactor!==void 0&&o!==Ot){let u=r.emissiveFactor;a.emissive=new Ge().setRGB(u[0],u[1],u[2],Wt)}return r.emissiveTexture!==void 0&&o!==Ot&&l.push(t.assignTexture(a,"emissiveMap",r.emissiveTexture,at)),Promise.all(l).then(function(){let u=new o(a);return r.name&&(u.name=r.name),ri(u,r),t.associations.set(u,{materials:e}),r.extensions&&ws(s,u,r),u})}createUniqueName(e){let t=xt.sanitizeNodeName(e||"");return t in this.nodeNamesUsed?t+"_"+ ++this.nodeNamesUsed[t]:(this.nodeNamesUsed[t]=0,t)}loadGeometries(e){let t=this,n=this.extensions,s=this.primitiveCache;function r(a){return n[it.KHR_DRACO_MESH_COMPRESSION].decodePrimitive(a,t).then(function(c){return wd(c,a,t)})}let o=[];for(let a=0,c=e.length;a<c;a++){let l=e[a],h=ky(l),u=s[h];if(u)o.push(u.promise);else{let f;l.extensions&&l.extensions[it.KHR_DRACO_MESH_COMPRESSION]?f=r(l):f=wd(new wt,l,t),s[h]={primitive:l,promise:f},o.push(f)}}return Promise.all(o)}loadMesh(e){let t=this,n=this.json,s=this.extensions,r=n.meshes[e],o=r.primitives,a=[];for(let c=0,l=o.length;c<l;c++){let h=o[c].material===void 0?Oy(this.cache):this.getDependency("material",o[c].material);a.push(h)}return a.push(t.loadGeometries(o)),Promise.all(a).then(function(c){let l=c.slice(0,c.length-1),h=c[c.length-1],u=[];for(let d=0,p=h.length;d<p;d++){let y=h[d],m=o[d],g,E=l[d];if(m.mode===wn.TRIANGLES||m.mode===wn.TRIANGLE_STRIP||m.mode===wn.TRIANGLE_FAN||m.mode===void 0)g=r.isSkinnedMesh===!0?new qr(y,E):new ht(y,E),g.isSkinnedMesh===!0&&g.normalizeSkinWeights(),m.mode===wn.TRIANGLE_STRIP?g.geometry=Ah(g.geometry,So):m.mode===wn.TRIANGLE_FAN&&(g.geometry=Ah(g.geometry,dr));else if(m.mode===wn.LINES)g=new Kr(y,E);else if(m.mode===wn.LINE_STRIP)g=new ms(y,E);else if(m.mode===wn.LINE_LOOP)g=new jr(y,E);else if(m.mode===wn.POINTS)g=new Jr(y,E);else throw new Error("THREE.GLTFLoader: Primitive mode unsupported: "+m.mode);Object.keys(g.geometry.morphAttributes).length>0&&By(g,r),g.name=t.createUniqueName(r.name||"mesh_"+e),ri(g,r),m.extensions&&ws(s,g,m),t.assignFinalMaterial(g),u.push(g)}for(let d=0,p=u.length;d<p;d++)t.associations.set(u[d],{meshes:e,primitives:d});if(u.length===1)return r.extensions&&ws(s,u[0],r),u[0];let f=new Ut;r.extensions&&ws(s,f,r),t.associations.set(f,{meshes:e});for(let d=0,p=u.length;d<p;d++)f.add(u[d]);return f})}loadCamera(e){let t,n=this.json.cameras[e],s=n[n.type];if(!s){console.warn("THREE.GLTFLoader: Missing camera parameters.");return}return n.type==="perspective"?t=new Ct(Ts.radToDeg(s.yfov),s.aspectRatio||1,s.znear||1,s.zfar||2e6):n.type==="orthographic"&&(t=new ti(-s.xmag,s.xmag,s.ymag,-s.ymag,s.znear,s.zfar)),n.name&&(t.name=this.createUniqueName(n.name)),ri(t,n),Promise.resolve(t)}loadSkin(e){let t=this.json.skins[e],n=[];for(let s=0,r=t.joints.length;s<r;s++)n.push(this._loadNodeShallow(t.joints[s]));return t.inverseBindMatrices!==void 0?n.push(this.getDependency("accessor",t.inverseBindMatrices)):n.push(null),Promise.all(n).then(function(s){let r=s.pop(),o=s,a=[],c=[];for(let l=0,h=o.length;l<h;l++){let u=o[l];if(u){a.push(u);let f=new Je;r!==null&&f.fromArray(r.array,l*16),c.push(f)}else console.warn('THREE.GLTFLoader: Joint "%s" could not be found.',t.joints[l])}return new Yr(a,c)})}loadAnimation(e){let t=this.json,n=this,s=t.animations[e],r=s.name?s.name:"animation_"+e,o=[],a=[],c=[],l=[],h=[];for(let u=0,f=s.channels.length;u<f;u++){let d=s.channels[u],p=s.samplers[d.sampler],y=d.target,m=y.node,g=s.parameters!==void 0?s.parameters[p.input]:p.input,E=s.parameters!==void 0?s.parameters[p.output]:p.output;y.node!==void 0&&(o.push(this.getDependency("node",m)),a.push(this.getDependency("accessor",g)),c.push(this.getDependency("accessor",E)),l.push(p),h.push(y))}return Promise.all([Promise.all(o),Promise.all(a),Promise.all(c),Promise.all(l),Promise.all(h)]).then(function(u){let f=u[0],d=u[1],p=u[2],y=u[3],m=u[4],g=[];for(let S=0,x=f.length;S<x;S++){let A=f[S],w=d[S],I=p[S],v=y[S],R=m[S];if(A===void 0)continue;A.updateMatrix&&A.updateMatrix();let F=n._createAnimationTracks(A,w,I,v,R);if(F)for(let N=0;N<F.length;N++)g.push(F[N])}let E=new co(r,void 0,g);return ri(E,s),E})}createNodeMesh(e){let t=this.json,n=this,s=t.nodes[e];return s.mesh===void 0?null:n.getDependency("mesh",s.mesh).then(function(r){let o=n._getNodeRef(n.meshCache,s.mesh,r);return s.weights!==void 0&&o.traverse(function(a){if(a.isMesh)for(let c=0,l=s.weights.length;c<l;c++)a.morphTargetInfluences[c]=s.weights[c]}),o})}loadNode(e){let t=this.json,n=this,s=t.nodes[e],r=n._loadNodeShallow(e),o=[],a=s.children||[];for(let l=0,h=a.length;l<h;l++)o.push(n.getDependency("node",a[l]));let c=s.skin===void 0?Promise.resolve(null):n.getDependency("skin",s.skin);return Promise.all([r,Promise.all(o),c]).then(function(l){let h=l[0],u=l[1],f=l[2];f!==null&&h.traverse(function(d){d.isSkinnedMesh&&d.bind(f,Hy)});for(let d=0,p=u.length;d<p;d++)h.add(u[d]);if(h.userData.pivot!==void 0&&u.length>0){let d=h.userData.pivot,p=u[0];h.pivot=new B().fromArray(d),h.position.x-=d[0],h.position.y-=d[1],h.position.z-=d[2],p.position.set(0,0,0),delete h.userData.pivot}return h})}_loadNodeShallow(e){let t=this.json,n=this.extensions,s=this;if(this.nodeCache[e]!==void 0)return this.nodeCache[e];let r=t.nodes[e],o=r.name?s.createUniqueName(r.name):"",a=[],c=s._invokeOne(function(l){return l.createNodeMesh&&l.createNodeMesh(e)});return c&&a.push(c),r.camera!==void 0&&a.push(s.getDependency("camera",r.camera).then(function(l){return s._getNodeRef(s.cameraCache,r.camera,l)})),s._invokeAll(function(l){return l.createNodeAttachment&&l.createNodeAttachment(e)}).forEach(function(l){a.push(l)}),this.nodeCache[e]=Promise.all(a).then(function(l){let h;if(r.isBone===!0?h=new Qs:l.length>1?h=new Ut:l.length===1?h=l[0]:h=new At,h!==l[0])for(let u=0,f=l.length;u<f;u++)h.add(l[u]);if(r.name&&(h.userData.name=r.name,h.name=o),ri(h,r),r.extensions&&ws(n,h,r),r.matrix!==void 0){let u=new Je;u.fromArray(r.matrix),h.applyMatrix4(u)}else r.translation!==void 0&&h.position.fromArray(r.translation),r.rotation!==void 0&&h.quaternion.fromArray(r.rotation),r.scale!==void 0&&h.scale.fromArray(r.scale);if(!s.associations.has(h))s.associations.set(h,{});else if(r.mesh!==void 0&&s.meshCache.refs[r.mesh]>1){let u=s.associations.get(h);s.associations.set(h,{...u})}return s.associations.get(h).nodes=e,h}),this.nodeCache[e]}loadScene(e){let t=this.extensions,n=this.json.scenes[e],s=this,r=new Ut;n.name&&(r.name=s.createUniqueName(n.name)),ri(r,n),n.extensions&&ws(t,r,n);let o=n.nodes||[],a=[];for(let c=0,l=o.length;c<l;c++)a.push(s.getDependency("node",o[c]));return Promise.all(a).then(function(c){for(let h=0,u=c.length;h<u;h++){let f=c[h];f.parent!==null?r.add(Md(f)):r.add(f)}let l=h=>{let u=new Map;for(let[f,d]of s.associations)(f instanceof on||f instanceof Tt)&&u.set(f,d);return h.traverse(f=>{let d=s.associations.get(f);d!=null&&u.set(f,d)}),u};return s.associations=l(r),r})}_createAnimationTracks(e,t,n,s,r){let o=[],a=e.name?e.name:e.uuid,c=[];function l(d){d.morphTargetInfluences&&c.push(d.name?d.name:d.uuid)}es[r.path]===es.weights?(l(e),e.isGroup&&e.children.forEach(l)):c.push(a);let h;switch(es[r.path]){case es.weights:h=vi;break;case es.rotation:h=bi;break;case es.translation:case es.scale:h=Xi;break;default:switch(n.itemSize){case 1:h=vi;break;case 2:case 3:default:h=Xi;break}break}let u=s.interpolation!==void 0?Uy[s.interpolation]:ds,f=this._getArrayFromAccessor(n);for(let d=0,p=c.length;d<p;d++){let y=new h(c[d]+"."+es[r.path],t.array,f,u);s.interpolation==="CUBICSPLINE"&&this._createCubicSplineTrackInterpolant(y),o.push(y)}return o}_getArrayFromAccessor(e){let t=e.array;if(e.normalized){let n=Qh(t.constructor),s=new Float32Array(t.length);for(let r=0,o=t.length;r<o;r++)s[r]=t[r]*n;t=s}return t}_createCubicSplineTrackInterpolant(e){e.createInterpolant=function(n){let s=this instanceof bi?Jh:Uc;return new s(this.times,this.values,this.getValueSize()/3,n)},e.createInterpolant.isInterpolantFactoryMethodGLTFCubicSpline=!0}};function Vy(i,e,t){let n=e.attributes,s=new Xt;if(n.POSITION!==void 0){let a=t.json.accessors[n.POSITION],c=a.min,l=a.max;if(c!==void 0&&l!==void 0){if(s.set(new B(c[0],c[1],c[2]),new B(l[0],l[1],l[2])),a.normalized){let h=Qh(br[a.componentType]);s.min.multiplyScalar(h),s.max.multiplyScalar(h)}}else{console.warn("THREE.GLTFLoader: Missing min/max properties for accessor POSITION.");return}}else return;let r=e.targets;if(r!==void 0){let a=new B,c=new B;for(let l=0,h=r.length;l<h;l++){let u=r[l];if(u.POSITION!==void 0){let f=t.json.accessors[u.POSITION],d=f.min,p=f.max;if(d!==void 0&&p!==void 0){if(c.setX(Math.max(Math.abs(d[0]),Math.abs(p[0]))),c.setY(Math.max(Math.abs(d[1]),Math.abs(p[1]))),c.setZ(Math.max(Math.abs(d[2]),Math.abs(p[2]))),f.normalized){let y=Qh(br[f.componentType]);c.multiplyScalar(y)}a.max(c)}else console.warn("THREE.GLTFLoader: Missing min/max properties for accessor POSITION.")}}s.expandByVector(a)}i.boundingBox=s;let o=new rn;s.getCenter(o.center),o.radius=s.min.distanceTo(s.max)/2,i.boundingSphere=o}function wd(i,e,t){let n=e.attributes,s=[];function r(o,a){return t.getDependency("accessor",o).then(function(c){i.setAttribute(a,c)})}for(let o in n){let a=$h[o]||o.toLowerCase();a in i.attributes||s.push(r(n[o],a))}if(e.indices!==void 0&&!i.index){let o=t.getDependency("accessor",e.indices).then(function(a){i.setIndex(a)});s.push(o)}return Qe.workingColorSpace!==Wt&&"COLOR_0"in n&&console.warn(`THREE.GLTFLoader: Converting vertex colors from "srgb-linear" to "${Qe.workingColorSpace}" not supported.`),ri(i,e),Vy(i,e,t),Promise.all(s).then(function(){return e.targets!==void 0?Fy(i,e.targets,t):i})}var Gy=Array.from({length:95},(i,e)=>String.fromCharCode(32+e)).join(""),Wy={src:"",ascii:!0,cellSize:10,cellAspect:.6,charset:Gy,colored:!0,color:"#ffffff",contrast:1.5,edgeContrast:3,exposure:1,invert:!1,background:"",highlight:"#066aff",environmentIntensity:1,roughness:-1,scale:3,xOffset:0,yOffset:0,floatIntensity:2,rotationIntensity:1,floatSpeed:2,orbit:!0,zoom:!1,autoRotate:!1,autoRotateSpeed:2,fov:65,cameraDistance:4.2,dracoDecoderPath:"https://www.gstatic.com/draco/versioned/decoders/1.5.7/",onLoad:null,onError:null},Cd=`
out vec2 vUv;
void main() {
  vUv = position.xy * 0.5 + 0.5;
  gl_Position = vec4(position.xy, 0.0, 1.0);
}`,Od=`
vec3 toSrgb(vec3 c) {
  c = clamp(c, 0.0, 1.0);
  return mix(c * 12.92, 1.055 * pow(c, vec3(1.0 / 2.4)) - 0.055, step(vec3(0.0031308), c));
}
`,Xy=`
precision highp float;
out vec4 outColor;
uniform sampler2D tScene;
uniform sampler2D tShapes;
uniform vec2 uResolution;
uniform vec2 uCellPx;
uniform int uGlyphCount;
uniform float uContrast;
uniform float uEdgeContrast;
uniform float uExposure;
uniform float uInvert;
${Od}
const vec2 INNER[6] = vec2[6](
  vec2(0.28, 0.26), vec2(0.72, 0.14),
  vec2(0.28, 0.56), vec2(0.72, 0.44),
  vec2(0.28, 0.86), vec2(0.72, 0.74)
);
const vec2 OUTER[10] = vec2[10](
  vec2(0.28, -0.2), vec2(0.72, -0.2),
  vec2(-0.22, 0.25), vec2(1.22, 0.25),
  vec2(-0.22, 0.5), vec2(1.22, 0.5),
  vec2(-0.22, 0.75), vec2(1.22, 0.75),
  vec2(0.28, 1.2), vec2(0.72, 1.2)
);
const vec2 RING[6] = vec2[6](
  vec2(1.0, 0.0), vec2(0.5, 0.8660254), vec2(-0.5, 0.8660254),
  vec2(-1.0, 0.0), vec2(-0.5, -0.8660254), vec2(0.5, -0.8660254)
);
vec2 cellBase;
vec4 fetchTap(vec2 p) {
  vec2 uv = p / uResolution;
  if (uv.x < 0.0 || uv.x > 1.0 || uv.y < 0.0 || uv.y > 1.0) return vec4(0.0);
  return texture(tScene, uv);
}
vec4 sampleCircle(vec2 c) {
  vec2 middle = cellBase + vec2(c.x, 1.0 - c.y) * uCellPx;
  float r = uCellPx.y * 0.161;
  vec4 acc = fetchTap(middle);
  for (int k = 0; k < 6; k++) acc += fetchTap(middle + RING[k] * r);
  return acc / 7.0;
}
float circleLum(vec4 acc) {
  vec3 straight = toSrgb(acc.rgb / max(acc.a, 1e-4));
  float level = clamp(dot(straight, vec3(0.2126, 0.7152, 0.0722)) * uExposure, 0.0, 1.0);
  level = mix(level, 1.0 - level, uInvert);
  return level * acc.a;
}
float dirContrast(float value, float ext) {
  float peak = max(value, ext);
  if (peak < 1e-4) return value;
  return pow(value / peak, uEdgeContrast) * peak;
}
void main() {
  cellBase = floor(gl_FragCoord.xy) * uCellPx;
  float v[6];
  vec3 colAcc = vec3(0.0);
  float alphaAcc = 0.0;
  for (int i = 0; i < 6; i++) {
    vec4 acc = sampleCircle(INNER[i]);
    v[i] = circleLum(acc);
    colAcc += acc.rgb;
    alphaAcc += acc.a;
  }
  float e[10];
  for (int i = 0; i < 10; i++) e[i] = circleLum(sampleCircle(OUTER[i]));
  v[0] = dirContrast(v[0], max(max(e[0], e[1]), max(e[2], e[4])));
  v[1] = dirContrast(v[1], max(max(e[0], e[1]), max(e[3], e[5])));
  v[2] = dirContrast(v[2], max(e[2], max(e[4], e[6])));
  v[3] = dirContrast(v[3], max(e[3], max(e[5], e[7])));
  v[4] = dirContrast(v[4], max(max(e[4], e[6]), max(e[8], e[9])));
  v[5] = dirContrast(v[5], max(max(e[5], e[7]), max(e[8], e[9])));
  float peak = max(max(max(v[0], v[1]), max(v[2], v[3])), max(v[4], v[5]));
  if (peak > 1e-4) {
    for (int i = 0; i < 6; i++) v[i] = pow(v[i] / peak, uContrast) * peak;
  }
  int best = 0;
  float bestD = 1e9;
  for (int g = 0; g < uGlyphCount; g++) {
    float d = 0.0;
    for (int i = 0; i < 6; i++) {
      float diff = v[i] - texelFetch(tShapes, ivec2(i, g), 0).r;
      d += diff * diff;
    }
    if (d < bestD) {
      bestD = d;
      best = g;
    }
  }
  vec3 cellColor = toSrgb(colAcc / max(alphaAcc, 1e-4));
  outColor = vec4(cellColor, float(best) / 255.0);
}`,qy=`
precision highp float;
in vec2 vUv;
out vec4 outColor;
uniform sampler2D tScene;
uniform sampler2D tCells;
uniform sampler2D tAtlas;
uniform vec2 uResolution;
uniform vec2 uCellPx;
uniform vec2 uGrid;
uniform vec2 uAtlasGrid;
uniform vec2 uAtlasPad;
uniform vec2 uAtlasInner;
uniform float uAscii;
uniform float uColored;
uniform vec3 uColor;
uniform vec3 uBackground;
uniform float uHasBg;
${Od}
void main() {
  if (uAscii < 0.5) {
    vec4 raw = texture(tScene, vUv);
    vec3 rawColor = toSrgb(raw.rgb);
    if (uHasBg > 0.5) {
      outColor = vec4(uBackground * (1.0 - raw.a) + rawColor, 1.0);
    } else {
      outColor = vec4(rawColor * raw.a, raw.a);
    }
    return;
  }
  vec2 fragCoord = vUv * uResolution;
  vec2 cellPos = fragCoord / uCellPx;
  vec2 cell = clamp(floor(cellPos), vec2(0.0), uGrid - 1.0);
  vec4 info = texelFetch(tCells, ivec2(cell), 0);
  float glyph = floor(info.a * 255.0 + 0.5);
  vec2 local = clamp(cellPos - cell, 0.0, 1.0);
  float gx = mod(glyph, uAtlasGrid.x);
  float gy = floor(glyph / uAtlasGrid.x);
  vec2 atlasUv = vec2(
    (gx + uAtlasPad.x + local.x * uAtlasInner.x) / uAtlasGrid.x,
    (uAtlasGrid.y - gy - 1.0 + uAtlasPad.y + local.y * uAtlasInner.y) /
      uAtlasGrid.y
  );
  vec2 atlasStep = uAtlasInner / uAtlasGrid;
  float mask = textureGrad(
    tAtlas,
    atlasUv,
    dFdx(cellPos) * atlasStep,
    dFdy(cellPos) * atlasStep
  ).a;
  vec3 glyphColor = mix(uColor, info.rgb, uColored);
  if (uHasBg > 0.5) {
    outColor = vec4(mix(uBackground, glyphColor, mask), 1.0);
  } else {
    outColor = vec4(glyphColor * mask, mask);
  }
}`,Yy=[{position:[-10.906,-1,1.846],rotation:[0,-.195,0],scale:[2.328,7.905,4.651]},{position:[-5.607,-.754,-.758],rotation:[0,.994,0],scale:[1.97,1.534,3.955]},{position:[6.167,-.16,7.803],rotation:[0,.561,0],scale:[3.927,6.285,3.687]},{position:[-2.017,.018,6.124],rotation:[0,.333,0],scale:[2.002,4.566,2.064]},{position:[2.291,-.756,-2.621],rotation:[0,-.286,0],scale:[1.546,1.552,1.496]},{position:[-2.193,-.369,-5.547],rotation:[0,.516,0],scale:[3.875,3.487,2.986]}],Zy=[{kind:"ring",intensity:15,position:[2,3,-2],scale:[10,10,10],lookAtCenter:!0},{kind:"box",intensity:80,position:[-14,10,8],scale:[.1,2.5,2.5]},{kind:"box",intensity:80,position:[-14,14,-4],scale:[.1,2.5,2.5],withLight:!0},{kind:"box",intensity:23,position:[14,12,0],scale:[.1,5,5],withLight:!0},{kind:"box",intensity:16,position:[0,9,14],scale:[5,5,.1],withLight:!0},{kind:"box",intensity:80,position:[7,8,-14],scale:[2.5,2.5,.1],withLight:!0},{kind:"box",intensity:80,position:[-7,16,-14],scale:[2.5,2.5,.1],withLight:!0},{kind:"box",intensity:1,position:[0,20,0],scale:[.1,.1,.1],withLight:!0},{kind:"box",intensity:20,position:[0,15,0],scale:[10,1,10],withLight:!0}],Pd=new B(0,-1,4).normalize(),tu=.3,Ro=2048,Ky=512,jy=127,Jy=1,$y=6,Qy=64,ev=.08,Id=.006,tv=64,Rn=8,nv=255,Ld=[[.28,.26],[.72,.14],[.28,.56],[.72,.44],[.28,.86],[.72,.74]];function Dd(i){return Math.min(Math.max(i||.6,.35),1.25)}function iv(i){let e=new Set([" "]),t=[" "];for(let n of i){if(t.length>=nv)break;n===`
`||n==="\r"||n==="	"||e.has(n)||(e.add(n),t.push(n))}return t}function sv(i,e,t,n,s){let r=new Float32Array(s*6),o=n*.26,a=t+Rn*2,c=n+Rn*2;for(let l=0;l<s;l++){let h=l%e*a+Rn,u=Math.floor(l/e)*c+Rn;for(let f=0;f<6;f++){let d=Ld[f][0]*t,p=Ld[f][1]*n,y=0,m=0;for(let g=Math.floor(p-o);g<=Math.ceil(p+o);g++)for(let E=Math.floor(d-o);E<=Math.ceil(d+o);E++){let S=E+.5-d,x=g+.5-p;S*S+x*x>o*o||(m+=1,!(E<-Rn||g<-Rn||E>=t+Rn||g>=n+Rn)&&(y+=i.data[((u+g)*i.width+h+E)*4+3]))}r[l*6+f]=m?y/(m*255):0}}for(let l=0;l<6;l++){let h=0;for(let u=0;u<s;u++)h=Math.max(h,r[u*6+l]);if(h>0)for(let u=0;u<s;u++)r[u*6+l]/=h}return r}function rv(i){if(i.length<4)return null;let e=(n,s)=>{for(let r=0;r<s.length;r++)if(i[n+r]!==s.charCodeAt(r))return!1;return!0};if(e(0,"glTF"))return"glb";if(i[0]===137&&e(1,"PNG")||i[0]===255&&i[1]===216||e(0,"RIFF")&&e(8,"WEBP")||e(0,"GIF8"))return"bitmap";let t="";try{t=new TextDecoder().decode(i.subarray(0,2048)).replace(/^\uFEFF/,"").trimStart()}catch{return null}return t.startsWith("{")?"gltf":t.startsWith("<")&&t.includes("<svg")?"svg":null}function Fd(i,e){let t=document.createElement("canvas");return t.width=Math.max(1,Math.round(i)),t.height=Math.max(1,Math.round(e)),t}function iu(i,e,t){let n=Fd(e,t),s=n.getContext("2d");if(!s)throw new Error("2d context unavailable");return s.drawImage(i,0,0,n.width,n.height),n}function ov(i){return new Promise((e,t)=>{let n=URL.createObjectURL(i),s=new Image;s.onload=()=>{URL.revokeObjectURL(n),e(s)},s.onerror=()=>{URL.revokeObjectURL(n),t(new Error("Could not decode the image"))},s.src=n})}async function av(i){if(typeof createImageBitmap!="function")return null;try{let e=await createImageBitmap(i),t=Math.max(e.width,e.height,1),n=Math.min(1,Ro/t),s=iu(e,e.width*n,e.height*n);return e.close(),s}catch{return null}}async function cv(i,e){let t=e==="svg";if(!t){let c=await av(i);if(c)return c}let n=await ov(i),s=n.naturalWidth||Ro,r=n.naturalHeight||Ro,o=Math.max(s,r,1),a=t?Ro/o:Math.min(1,Ro/o);return iu(n,s*a,r*a)}function lv(i,e,t){let n=[];for(let h=0;h<t-1;h++)for(let u=0;u<e-1;u++){let f=h*e+u,d=i[f]|i[f+1]<<1|i[f+e+1]<<2|i[f+e]<<3;if(d===0||d===15)continue;let p=u+.5,y=h+.5;switch(d){case 1:case 14:n.push(u,y,p,h);break;case 2:case 13:n.push(p,h,u+1,y);break;case 3:case 12:n.push(u,y,u+1,y);break;case 4:case 11:n.push(u+1,y,p,h+1);break;case 6:case 9:n.push(p,h,p,h+1);break;case 7:case 8:n.push(u,y,p,h+1);break;case 5:n.push(u,y,p,h,u+1,y,p,h+1);break;default:n.push(p,h,u+1,y,u,y,p,h+1);break}}let s=n.length/4,r=e*2+1,o=new Map,a=h=>n[h*2+1]*2*r+n[h*2]*2;for(let h=0;h<s;h++)for(let u of[h*2,h*2+1]){let f=a(u),d=o.get(f);d?d.push(h):o.set(f,[h])}let c=new Uint8Array(s),l=[];for(let h=0;h<s;h++){if(c[h])continue;let u=[],f=h,d=n[h*4],p=n[h*4+1];for(;f>=0&&!c[f];){c[f]=1;let y=f*4,m=n[y]===d&&n[y+1]===p;d=m?n[y+2]:n[y],p=m?n[y+3]:n[y+1],u.push(d,p);let g=o.get(p*2*r+d*2),E=-1;if(g){for(let S of g)if(!c[S]){E=S;break}}f=E}u.length>=8&&l.push(u)}return l}function hv(i,e){let t=i.length/2;if(t<4)return i;let n=new Uint8Array(t);n[0]=1,n[t-1]=1;let s=[0,t-1],r=e*e;for(;s.length;){let a=s.pop(),c=s.pop();if(a-c<2)continue;let l=i[c*2],h=i[c*2+1],u=i[a*2]-l,f=i[a*2+1]-h,d=u*u+f*f,p=-1,y=r;for(let m=c+1;m<a;m++){let g=i[m*2]-l,E=i[m*2+1]-h,S=d>0?(g*u+E*f)/d:0,x=S<0?0:S>1?1:S,A=g-u*x,w=E-f*x,I=A*A+w*w;I>y&&(p=m,y=I)}p<0||(n[p]=1,s.push(c,p,p,a))}let o=[];for(let a=0;a<t;a++)n[a]&&o.push(i[a*2],i[a*2+1]);return o}function Nd(i){let e=0;for(let t=0,n=i.length-2;t<i.length;n=t,t+=2)e+=(i[n]-i[t])*(i[n+1]+i[t+1]);return Math.abs(e)/2}function Ud(i,e,t){let n=!1;for(let s=0,r=i.length-2;s<i.length;r=s,s+=2){let o=i[s+1],a=i[r+1];if(o>t==a>t)continue;let c=(t-o)/(a-o);e<i[s]+c*(i[r]-i[s])&&(n=!n)}return n}function uv(i,e,t){let n=()=>new kn([new de(0,0),new de(e,0),new de(e,t),new de(0,t)]),s=Math.min(1,Ky/Math.max(i.width,i.height,1)),r=s<1?iu(i,i.width*s,i.height*s):i,o=r.getContext("2d",{willReadFrequently:!0});if(!o)return[n()];let a=r.width,c=r.height,l=o.getImageData(0,0,a,c).data,h=a+2,u=c+2,f=new Uint8Array(h*u),d=0;for(let E=0;E<c;E++)for(let S=0;S<a;S++){let x=l[(E*a+S)*4+3]>=jy?1:0;f[(E+1)*h+S+1]=x,d+=x}if(d>=a*c*.995)return[n()];let p=lv(f,h,u).map(E=>hv(E,Jy)).filter(E=>E.length>=6&&Nd(E)>=$y).map(E=>({points:E,area:Nd(E),depth:0})).sort((E,S)=>S.area-E.area).slice(0,Qy);if(!p.length)return[n()];for(let E of p)for(let S of p)S!==E&&S.area>E.area&&Ud(S.points,E.points[0],E.points[1])&&(E.depth+=1);let y=E=>{let S=[];for(let x=0;x<E.length;x+=2)S.push(new de((E[x]-.5)/a*e,(1-(E[x+1]-.5)/c)*t));return S},m=new Map;for(let E of p)E.depth%2===0&&m.set(E,new kn(y(E.points)));for(let E of p){if(E.depth%2===0)continue;let S=null;for(let A of p)A.depth===E.depth-1&&Ud(A.points,E.points[0],E.points[1])&&(!S||A.area<S.area)&&(S=A);let x=S?m.get(S):void 0;x&&x.holes.push(new tn(y(E.points)))}let g=[...m.values()];return g.length?g:[n()]}function fv(i,e){let t=Math.max(i.width,i.height,1),n=i.width/t,s=i.height/t,r=new _s(uv(i,n,s),{depth:ev,bevelEnabled:!0,bevelThickness:Id,bevelSize:Id,bevelOffset:0,bevelSegments:2,steps:1,curveSegments:1}),o=r.getAttribute("position"),a=new Float32Array(o.count*2);for(let h=0;h<o.count;h++)a[h*2]=o.getX(h)/n,a[h*2+1]=o.getY(h)/s;r.setAttribute("uv",new yt(a,2));let c=new Qn(i);c.colorSpace=at,c.anisotropy=e;let l=new an({map:c,roughness:.6,metalness:0});return new ht(r,l)}function nu(i){i.traverse(e=>{let t=e;t.geometry&&t.geometry.dispose();let n=Array.isArray(t.material)?t.material:[t.material];for(let s of n)if(s){for(let r of Object.values(s))r instanceof Tt&&r.dispose();s.dispose()}})}function Bd(i,e={}){let{canvas:t}=i,n={...Wy,...e},s;try{s=new _r({canvas:t,antialias:!1,alpha:!0,powerPreference:"high-performance"})}catch{return null}s.toneMapping=bs,s.setClearColor(0,0);let r=new Bn,o=new Ct(n.fov,1,.1,200);o.position.copy(Pd).multiplyScalar(n.cameraDistance);let a=new Ut;a.position.y=tu;let c=new Ut;a.add(c),r.add(a);let l=new yr(o,t);l.enableDamping=!0,l.enablePan=!1;let h=new jt(1,1,{samples:4});h.texture.colorSpace=at;let u=new de(1,1),f=new de(6,10),d=new de(1,1),p=new jt(1,1,{depthBuffer:!1,stencilBuffer:!1,minFilter:St,magFilter:St}),y=new Jt({glslVersion:pr,vertexShader:Cd,fragmentShader:qy,uniforms:{tScene:{value:h.texture},tCells:{value:p.texture},tAtlas:{value:null},uResolution:{value:u},uCellPx:{value:f},uGrid:{value:d},uAtlasGrid:{value:new de(1,1)},uAtlasPad:{value:new de(0,0)},uAtlasInner:{value:new de(1,1)},uAscii:{value:1},uColored:{value:1},uColor:{value:new Ge(1,1,1)},uBackground:{value:new Ge(0,0,0)},uHasBg:{value:0}},depthTest:!1,depthWrite:!1,blending:xn}),m=new wt;m.setAttribute("position",new yt(new Float32Array([-1,-1,0,3,-1,0,-1,3,0]),3));let g=new ht(m,y);g.frustumCulled=!1;let E=new Bn;E.add(g);let S=new ti(-1,1,1,-1,0,1),x=new Jt({glslVersion:pr,vertexShader:Cd,fragmentShader:Xy,uniforms:{tScene:{value:h.texture},tShapes:{value:null},uResolution:{value:u},uCellPx:{value:f},uGlyphCount:{value:1},uContrast:{value:1.5},uEdgeContrast:{value:3},uExposure:{value:1},uInvert:{value:0}},depthTest:!1,depthWrite:!1,blending:xn}),A=new ht(m,x);A.frustumCulled=!1;let w=new Bn;w.add(A);let I=null,v=null,R=null,F=0,N=new Qi(s),H=null,oe=null,ie=null,X=!0;function ee(){H=new Bn;let T=new Ut;T.position.set(0,-.5,0),H.add(T);for(let[_e,Se]of[[-15,15],[15,15],[15,-15],[-15,-15]]){let ae=new Si(16777215,2,0,.2,1,0);ae.position.set(_e,20,Se),T.add(ae,ae.target)}let _=new Hn(16777215,100,28,2);_.position.set(.5,14,.5),T.add(_);let O=new En,W=new ht(O,new an({color:"gray",side:Ft}));W.position.set(0,13.2,0),W.scale.set(31.5,28.5,31.5),T.add(W);let Q=new an({color:16777215});for(let _e of Yy){let Se=new ht(O,Q);Se.position.set(..._e.position),Se.rotation.set(..._e.rotation),Se.scale.set(..._e.scale),T.add(Se)}for(let _e of Zy){let Se=_e.kind==="ring"?new ys(.5,1,64):new En,ae=new Ot({side:Bt,toneMapped:!1});ae.color.set(_e.kind==="ring"?n.highlight:"#ffffff").multiplyScalar(_e.intensity),_e.kind==="ring"&&(oe=ae);let fe=new ht(Se,ae);if(fe.position.set(..._e.position),fe.scale.set(..._e.scale),_e.lookAtCenter&&fe.lookAt(0,0,0),T.add(fe),_e.withLight){let J=new Hn(16777215,100,28,2);J.position.set(..._e.position),T.add(J)}}}function Y(){H||ee(),oe&&oe.color.set(n.highlight).multiplyScalar(15),ie?.dispose(),ie=N.fromScene(H,0,.1,1e3),r.environment=ie.texture}let G=null,ge=1,ye=null,Me=0,Re=!1,ze=new Mr,qe=new vr;qe.setDecoderPath(n.dracoDecoderPath),ze.setDRACOLoader(qe);function We(){G&&G.traverse(T=>{let _=T,O=Array.isArray(_.material)?_.material:[_.material];for(let W of O){let Q=W;!Q||typeof Q.roughness!="number"||(Q.userData.baseRoughness===void 0&&(Q.userData.baseRoughness=Q.roughness),Q.roughness=n.roughness>=0?n.roughness:Q.userData.baseRoughness)}})}function pe(){G&&c.scale.setScalar(n.scale/ge)}function q(){G&&(c.remove(G),nu(G),G=null)}function U(T){q(),G=T;let _=new Xt().setFromObject(G),O=_.getSize(new B),W=_.getCenter(new B);ge=Math.max(O.x,O.y,O.z,1e-4),G.position.sub(W),We(),pe(),c.add(G)}async function L(){let T=n.src;if(T===ye)return;ye=T;let _=++Me;if(!T){q();return}try{let O=await fetch(T);if(!O.ok)throw new Error(`HTTP ${O.status}`);let W=await O.arrayBuffer();if(Re||_!==Me)return;let Q=new Uint8Array(W),_e=rv(Q);if(!_e)throw new Error("Unrecognized asset format");if(_e==="glb"||_e==="gltf"){qe.setDecoderPath(n.dracoDecoderPath);let Se=T.slice(0,T.lastIndexOf("/")+1),ae=_e==="glb"?W:new TextDecoder().decode(Q),fe=await ze.parseAsync(ae,Se);if(Re||_!==Me){nu(fe.scene);return}U(fe.scene)}else{let Se=new Blob([W],{type:_e==="svg"?"image/svg+xml":""}),ae=await cv(Se,_e);if(Re||_!==Me)return;U(fv(ae,s.capabilities.getMaxAnisotropy()))}n.onLoad?.()}catch(O){if(Re||_!==Me)return;n.onError?.(O)}}let P=window.matchMedia("(prefers-reduced-motion: reduce)"),z=P.matches,he=()=>{z=P.matches,z&&a.rotation.set(0,0,0),V()};P.addEventListener("change",he);function me(){let T=Dd(n.cellAspect);if(R===n.charset&&F===T)return;let _=iv(n.charset),O=tv,W=Math.max(Math.round(O*T),8),Q=W+Rn*2,_e=O+Rn*2,Se=Math.ceil(Math.sqrt(_.length)),ae=Math.ceil(_.length/Se),fe=Fd(Se*Q,ae*_e),J=fe.getContext("2d");if(!J)return;R=n.charset,F=T,J.clearRect(0,0,fe.width,fe.height),J.fillStyle="#ffffff",J.textAlign="center",J.textBaseline="middle";let xe=Math.floor(Math.min(O*.92,W/.58));J.font=`600 ${xe}px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace`;for(let Ee=0;Ee<_.length;Ee++)J.fillText(_[Ee],Ee%Se*Q+Q/2,Math.floor(Ee/Se)*_e+_e/2);let ue=J.getImageData(0,0,fe.width,fe.height),ve=sv(ue,Se,W,O,_.length);I?.dispose(),v?.dispose(),I=new Qn(fe),I.minFilter=yn,I.magFilter=bt,I.wrapS=Gt,I.wrapT=Gt,v=new Gi(ve,6,_.length,fr,nn),v.needsUpdate=!0,y.uniforms.tAtlas.value=I,y.uniforms.uAtlasGrid.value.set(Se,ae),y.uniforms.uAtlasPad.value.set(Rn/Q,Rn/_e),y.uniforms.uAtlasInner.value.set(W/Q,O/_e),x.uniforms.tShapes.value=v,x.uniforms.uGlyphCount.value=_.length}function C(){let T=s.getPixelRatio(),_=Math.max(n.cellSize,3)*T,O=_*Dd(n.cellAspect);f.set(O,_);let W=Math.max(Math.ceil(u.x/O),1),Q=Math.max(Math.ceil(u.y/_),1);d.set(W,Q),(p.width!==W||p.height!==Q)&&p.setSize(W,Q)}function V(){r.environmentIntensity=n.environmentIntensity,l.enableRotate=n.orbit,l.enableZoom=n.zoom,l.autoRotate=n.autoRotate&&!z,l.autoRotateSpeed=n.autoRotateSpeed,o.fov=n.fov,o.updateProjectionMatrix(),a.position.x=n.xOffset,a.position.y=tu+n.yOffset,x.uniforms.uContrast.value=Math.max(n.contrast,.05),x.uniforms.uEdgeContrast.value=Math.max(n.edgeContrast,.05),x.uniforms.uExposure.value=Math.max(n.exposure,0),x.uniforms.uInvert.value=n.invert?1:0,y.uniforms.uAscii.value=n.ascii?1:0,y.uniforms.uColored.value=n.colored?1:0,y.uniforms.uColor.value.setStyle(n.color||"#ffffff",An),y.uniforms.uHasBg.value=n.background?1:0,n.background&&y.uniforms.uBackground.value.setStyle(n.background,An),me(),C(),We(),pe()}function D(){let T=Math.max(t.clientWidth,1),_=Math.max(t.clientHeight,1),O=Math.min(window.devicePixelRatio||1,2);s.setPixelRatio(O),s.setSize(T,_,!1);let W=Math.round(T*O),Q=Math.round(_*O);h.setSize(W,Q),u.set(W,Q),o.aspect=T/_,o.updateProjectionMatrix(),C()}let j=new ResizeObserver(D);j.observe(t),D(),V(),L();let $=!0,le=!1;function te(T){if(!$){we=0,k();return}let _=we?Math.min((T-we)/1e3,.1):0;we=T,X&&(X=!1,Y()),l.update(),z||(Ae+=_*n.floatSpeed,a.rotation.x=Math.cos(Ae/4)/8*n.rotationIntensity,a.rotation.y=Math.sin(Ae/4)/8*n.rotationIntensity,a.rotation.z=Math.sin(Ae/4)/20*n.rotationIntensity,a.position.y=tu+n.yOffset+Math.sin(Ae/1.5)/10*n.floatIntensity),s.setRenderTarget(h),s.render(r,o),n.ascii&&(s.setRenderTarget(p),s.render(w,S)),s.setRenderTarget(null),s.render(E,S)}function ne(){le||!$||Re||(le=!0,s.setAnimationLoop(te))}function k(){le&&(le=!1,s.setAnimationLoop(null))}let b=typeof IntersectionObserver<"u"?new IntersectionObserver(T=>{$=T[T.length-1]?.isIntersecting??!0,$?ne():k()}):null;b?.observe(t);let we=0,Ae=Math.random()*100;return ne(),{setOptions(T){let _=!1;for(let[Q,_e]of Object.entries(T))if(typeof _e!="function"&&n[Q]!==_e){_=!0;break}if(!_){Object.assign(n,T);return}let O=n.highlight,W=n.cameraDistance;Object.assign(n,T),n.highlight!==O&&(X=!0),n.cameraDistance!==W&&o.position.copy(Pd).multiplyScalar(n.cameraDistance),V(),D(),L(),ne()},resize:D,destroy(){Re=!0,Me+=1,k(),j.disconnect(),b?.disconnect(),P.removeEventListener("change",he),l.dispose(),q(),H&&nu(H),ie?.dispose(),N.dispose(),qe.dispose(),h.dispose(),p.dispose(),x.dispose(),I?.dispose(),v?.dispose(),m.dispose(),y.dispose(),s.dispose()}}}var Co=at,Sr=class i extends ln{constructor(e){super(e),this.defaultDPI=90,this.defaultUnit="px"}load(e,t,n,s){let r=this,o=new zn(r.manager);o.setPath(r.path),o.setRequestHeader(r.requestHeader),o.setWithCredentials(r.withCredentials),o.load(e,function(a){try{t(r.parse(a))}catch(c){s?s(c):console.error(c),r.manager.itemError(e)}},n,s)}parse(e){let t=this;function n(U,L){if(U.nodeType!==1)return;U.hasAttribute("filter")&&console.warn("THREE.SVGLoader: Filters are not supported.");let P=A(U),z=!1,he=null;switch(U.nodeName){case"svg":L=y(U,L);break;case"style":r(U);break;case"g":L=y(U,L);break;case"path":L=y(U,L),U.hasAttribute("d")&&(he=s(U));break;case"rect":L=y(U,L),he=c(U);break;case"polygon":L=y(U,L),he=l(U);break;case"polyline":L=y(U,L),he=h(U);break;case"circle":L=y(U,L),he=u(U);break;case"ellipse":L=y(U,L),he=f(U);break;case"line":L=y(U,L),he=d(U);break;case"defs":z=!0;break;case"use":L=y(U,L);let V=(U.getAttributeNS("http://www.w3.org/1999/xlink","href")||"").substring(1),D=U.viewportElement.getElementById(V);D?n(D,L):console.warn("SVGLoader: 'use node' references non-existent node id: "+V);break;default:}if(he){L.fill!==void 0&&L.fill!=="none"&&!L.fill.startsWith("url")&&he.color.setStyle(L.fill,Co),v(he,We),X.push(he);let C=Object.assign({},L);C.strokeWidth=L.strokeWidth*oe(We),he.userData={node:U,style:C,transform:We.clone(),gradients:Y}}let me=U.childNodes;for(let C=0;C<me.length;C++){let V=me[C];z&&V.nodeName!=="style"&&V.nodeName!=="defs"||n(V,L)}P&&(G.pop(),G.length>0?We.copy(G[G.length-1]):We.identity())}function s(U){let L=new Vn,P=new de,z=new de,he=new de,me=!0,C=!1,V=U.getAttribute("d");if(V===""||V==="none")return null;let D=V.match(/[a-df-z][^a-df-z]*/ig);for(let j=0,$=D.length;j<$;j++){let le=D[j],te=le.charAt(0),ne=le.slice(1).trim();me===!0&&(C=!0,me=!1);let k;switch(te){case"M":k=g(ne);for(let b=0,we=k.length;b<we;b+=2)P.x=k[b+0],P.y=k[b+1],z.x=P.x,z.y=P.y,b===0?L.moveTo(P.x,P.y):L.lineTo(P.x,P.y),b===0&&he.copy(P);break;case"H":k=g(ne);for(let b=0,we=k.length;b<we;b++)P.x=k[b],z.x=P.x,z.y=P.y,L.lineTo(P.x,P.y),b===0&&C===!0&&he.copy(P);break;case"V":k=g(ne);for(let b=0,we=k.length;b<we;b++)P.y=k[b],z.x=P.x,z.y=P.y,L.lineTo(P.x,P.y),b===0&&C===!0&&he.copy(P);break;case"L":k=g(ne);for(let b=0,we=k.length;b<we;b+=2)P.x=k[b+0],P.y=k[b+1],z.x=P.x,z.y=P.y,L.lineTo(P.x,P.y),b===0&&C===!0&&he.copy(P);break;case"C":k=g(ne);for(let b=0,we=k.length;b<we;b+=6)L.bezierCurveTo(k[b+0],k[b+1],k[b+2],k[b+3],k[b+4],k[b+5]),z.x=k[b+2],z.y=k[b+3],P.x=k[b+4],P.y=k[b+5],b===0&&C===!0&&he.copy(P);break;case"S":k=g(ne);for(let b=0,we=k.length;b<we;b+=4)L.bezierCurveTo(m(P.x,z.x),m(P.y,z.y),k[b+0],k[b+1],k[b+2],k[b+3]),z.x=k[b+0],z.y=k[b+1],P.x=k[b+2],P.y=k[b+3],b===0&&C===!0&&he.copy(P);break;case"Q":k=g(ne);for(let b=0,we=k.length;b<we;b+=4)L.quadraticCurveTo(k[b+0],k[b+1],k[b+2],k[b+3]),z.x=k[b+0],z.y=k[b+1],P.x=k[b+2],P.y=k[b+3],b===0&&C===!0&&he.copy(P);break;case"T":k=g(ne);for(let b=0,we=k.length;b<we;b+=2){let Ae=m(P.x,z.x),T=m(P.y,z.y);L.quadraticCurveTo(Ae,T,k[b+0],k[b+1]),z.x=Ae,z.y=T,P.x=k[b+0],P.y=k[b+1],b===0&&C===!0&&he.copy(P)}break;case"A":k=g(ne,[3,4],7);for(let b=0,we=k.length;b<we;b+=7){if(k[b+5]==P.x&&k[b+6]==P.y)continue;let Ae=P.clone();P.x=k[b+5],P.y=k[b+6],z.x=P.x,z.y=P.y,o(L,k[b],k[b+1],k[b+2],k[b+3],k[b+4],Ae,P),b===0&&C===!0&&he.copy(P)}break;case"m":k=g(ne);for(let b=0,we=k.length;b<we;b+=2)P.x+=k[b+0],P.y+=k[b+1],z.x=P.x,z.y=P.y,b===0?L.moveTo(P.x,P.y):L.lineTo(P.x,P.y),b===0&&he.copy(P);break;case"h":k=g(ne);for(let b=0,we=k.length;b<we;b++)P.x+=k[b],z.x=P.x,z.y=P.y,L.lineTo(P.x,P.y),b===0&&C===!0&&he.copy(P);break;case"v":k=g(ne);for(let b=0,we=k.length;b<we;b++)P.y+=k[b],z.x=P.x,z.y=P.y,L.lineTo(P.x,P.y),b===0&&C===!0&&he.copy(P);break;case"l":k=g(ne);for(let b=0,we=k.length;b<we;b+=2)P.x+=k[b+0],P.y+=k[b+1],z.x=P.x,z.y=P.y,L.lineTo(P.x,P.y),b===0&&C===!0&&he.copy(P);break;case"c":k=g(ne);for(let b=0,we=k.length;b<we;b+=6)L.bezierCurveTo(P.x+k[b+0],P.y+k[b+1],P.x+k[b+2],P.y+k[b+3],P.x+k[b+4],P.y+k[b+5]),z.x=P.x+k[b+2],z.y=P.y+k[b+3],P.x+=k[b+4],P.y+=k[b+5],b===0&&C===!0&&he.copy(P);break;case"s":k=g(ne);for(let b=0,we=k.length;b<we;b+=4)L.bezierCurveTo(m(P.x,z.x),m(P.y,z.y),P.x+k[b+0],P.y+k[b+1],P.x+k[b+2],P.y+k[b+3]),z.x=P.x+k[b+0],z.y=P.y+k[b+1],P.x+=k[b+2],P.y+=k[b+3],b===0&&C===!0&&he.copy(P);break;case"q":k=g(ne);for(let b=0,we=k.length;b<we;b+=4)L.quadraticCurveTo(P.x+k[b+0],P.y+k[b+1],P.x+k[b+2],P.y+k[b+3]),z.x=P.x+k[b+0],z.y=P.y+k[b+1],P.x+=k[b+2],P.y+=k[b+3],b===0&&C===!0&&he.copy(P);break;case"t":k=g(ne);for(let b=0,we=k.length;b<we;b+=2){let Ae=m(P.x,z.x),T=m(P.y,z.y);L.quadraticCurveTo(Ae,T,P.x+k[b+0],P.y+k[b+1]),z.x=Ae,z.y=T,P.x=P.x+k[b+0],P.y=P.y+k[b+1],b===0&&C===!0&&he.copy(P)}break;case"a":k=g(ne,[3,4],7);for(let b=0,we=k.length;b<we;b+=7){if(k[b+5]==0&&k[b+6]==0)continue;let Ae=P.clone();P.x+=k[b+5],P.y+=k[b+6],z.x=P.x,z.y=P.y,o(L,k[b],k[b+1],k[b+2],k[b+3],k[b+4],Ae,P),b===0&&C===!0&&he.copy(P)}break;case"Z":case"z":L.currentPath.autoClose=!0,L.currentPath.curves.length>0&&(P.copy(he),L.currentPath.currentPoint.copy(P),me=!0);break;default:console.warn(le)}C=!1}return L}function r(U){if(!(!U.sheet||!U.sheet.cssRules||!U.sheet.cssRules.length))for(let L=0;L<U.sheet.cssRules.length;L++){let P=U.sheet.cssRules[L];if(P.type!==1)continue;let z=P.selectorText.split(/,/gm).filter(Boolean).map(he=>he.trim());for(let he=0;he<z.length;he++){let me=Object.fromEntries(Object.entries(P.style).filter(([,C])=>C!==""));ee[z[he]]=Object.assign(ee[z[he]]||{},me)}}}function o(U,L,P,z,he,me,C,V){if(L==0||P==0){U.lineTo(V.x,V.y);return}z=z*Math.PI/180,L=Math.abs(L),P=Math.abs(P);let D=(C.x-V.x)/2,j=(C.y-V.y)/2,$=Math.cos(z)*D+Math.sin(z)*j,le=-Math.sin(z)*D+Math.cos(z)*j,te=L*L,ne=P*P,k=$*$,b=le*le,we=k/te+b/ne;if(we>1){let fe=Math.sqrt(we);L=fe*L,P=fe*P,te=L*L,ne=P*P}let Ae=te*b+ne*k,T=(te*ne-Ae)/Ae,_=Math.sqrt(Math.max(0,T));he===me&&(_=-_);let O=_*L*le/P,W=-_*P*$/L,Q=Math.cos(z)*O-Math.sin(z)*W+(C.x+V.x)/2,_e=Math.sin(z)*O+Math.cos(z)*W+(C.y+V.y)/2,Se=a(1,0,($-O)/L,(le-W)/P),ae=a(($-O)/L,(le-W)/P,(-$-O)/L,(-le-W)/P)%(Math.PI*2);U.currentPath.absellipse(Q,_e,L,P,Se,Se+ae,me===0,z)}function a(U,L,P,z){let he=U*P+L*z,me=Math.sqrt(U*U+L*L)*Math.sqrt(P*P+z*z),C=Math.acos(Math.max(-1,Math.min(1,he/me)));return U*z-L*P<0&&(C=-C),C}function c(U){let L=x(U.getAttribute("x")||0),P=x(U.getAttribute("y")||0),z=x(U.getAttribute("rx")||U.getAttribute("ry")||0),he=x(U.getAttribute("ry")||U.getAttribute("rx")||0),me=x(U.getAttribute("width")),C=x(U.getAttribute("height")),V=1-.551915024494,D=new Vn;return D.moveTo(L+z,P),D.lineTo(L+me-z,P),(z!==0||he!==0)&&D.bezierCurveTo(L+me-z*V,P,L+me,P+he*V,L+me,P+he),D.lineTo(L+me,P+C-he),(z!==0||he!==0)&&D.bezierCurveTo(L+me,P+C-he*V,L+me-z*V,P+C,L+me-z,P+C),D.lineTo(L+z,P+C),(z!==0||he!==0)&&D.bezierCurveTo(L+z*V,P+C,L,P+C-he*V,L,P+C-he),D.lineTo(L,P+he),(z!==0||he!==0)&&D.bezierCurveTo(L,P+he*V,L+z*V,P,L+z,P),D}function l(U){function L(me,C,V){let D=x(C),j=x(V);he===0?z.moveTo(D,j):z.lineTo(D,j),he++}let P=/([+-]?\d*\.?\d+(?:e[+-]?\d+)?)(?:,|\s)([+-]?\d*\.?\d+(?:e[+-]?\d+)?)/g,z=new Vn,he=0;return U.getAttribute("points").replace(P,L),z.currentPath.autoClose=!0,z}function h(U){function L(me,C,V){let D=x(C),j=x(V);he===0?z.moveTo(D,j):z.lineTo(D,j),he++}let P=/([+-]?\d*\.?\d+(?:e[+-]?\d+)?)(?:,|\s)([+-]?\d*\.?\d+(?:e[+-]?\d+)?)/g,z=new Vn,he=0;return U.getAttribute("points").replace(P,L),z.currentPath.autoClose=!1,z}function u(U){let L=x(U.getAttribute("cx")||0),P=x(U.getAttribute("cy")||0),z=x(U.getAttribute("r")||0),he=new tn;he.absarc(L,P,z,0,Math.PI*2);let me=new Vn;return me.subPaths.push(he),me}function f(U){let L=x(U.getAttribute("cx")||0),P=x(U.getAttribute("cy")||0),z=x(U.getAttribute("rx")||0),he=x(U.getAttribute("ry")||0),me=new tn;me.absellipse(L,P,z,he,0,Math.PI*2);let C=new Vn;return C.subPaths.push(me),C}function d(U){let L=x(U.getAttribute("x1")||0),P=x(U.getAttribute("y1")||0),z=x(U.getAttribute("x2")||0),he=x(U.getAttribute("y2")||0),me=new Vn;return me.moveTo(L,P),me.lineTo(z,he),me.currentPath.autoClose=!1,me}function p(U){let L="http://www.w3.org/1999/xlink",P=U.querySelectorAll("linearGradient, radialGradient"),z=["x1","y1","x2","y2","cx","cy","r","fx","fy","gradientUnits","gradientTransform","spreadMethod"],he={};for(let C of P){let V=C.getAttribute("id");if(!V)continue;let D={type:C.nodeName==="radialGradient"?"radialGradient":"linearGradient",attrs:{},stops:null,href:null},j=C.getAttributeNS(L,"href")||C.getAttribute("href")||"";j.startsWith("#")&&(D.href=j.substring(1));for(let le of z)C.hasAttribute(le)&&(D.attrs[le]=C.getAttribute(le));let $=C.querySelectorAll("stop");if($.length>0){D.stops=[];for(let le of $){let te=le.getAttribute("stop-color");!te&&le.style&&(te=le.style["stop-color"]),te||(te="#000");let ne=le.getAttribute("stop-opacity");(ne===null||ne==="")&&le.style&&(ne=le.style["stop-opacity"]),ne=ne===null||ne===""||ne===void 0?1:Math.max(0,Math.min(1,parseFloat(ne)));let k=Math.max(0,Math.min(1,parseFloat(le.getAttribute("offset")||"0")));D.stops.push({offset:k,color:te,opacity:ne})}}he[V]=D}function me(C,V){let D=he[C];if(!D||V.has(C))return D;if(V.add(C),D.href&&he[D.href]){let j=me(D.href,V);if(j){D.stops||(D.stops=j.stops);for(let $ in j.attrs)$ in D.attrs||(D.attrs[$]=j.attrs[$])}}return D}for(let C in he)me(C,new Set);for(let C in he){let le=function(te){return typeof te!="string"?0:te.endsWith("%")?parseFloat(te)/100:x(te)},V=he[C],D=V.attrs,j=D.gradientUnits==="userSpaceOnUse"?"userSpaceOnUse":"objectBoundingBox",$={type:V.type,gradientUnits:j,spreadMethod:D.spreadMethod==="reflect"||D.spreadMethod==="repeat"?D.spreadMethod:"pad",gradientTransform:null,stops:(V.stops||[]).slice().sort((te,ne)=>te.offset-ne.offset)};if(D.gradientTransform&&($.gradientTransform=new Ze,I(D.gradientTransform,$.gradientTransform)),V.type==="linearGradient")$.x1=D.x1!==void 0?le(D.x1):0,$.y1=D.y1!==void 0?le(D.y1):0,$.x2=D.x2!==void 0?le(D.x2):j==="objectBoundingBox"?1:0,$.y2=D.y2!==void 0?le(D.y2):0;else{let te=j==="objectBoundingBox"?.5:0,ne=j==="objectBoundingBox"?.5:0;$.cx=D.cx!==void 0?le(D.cx):te,$.cy=D.cy!==void 0?le(D.cy):te,$.r=D.r!==void 0?le(D.r):ne,$.fx=D.fx!==void 0?le(D.fx):$.cx,$.fy=D.fy!==void 0?le(D.fy):$.cy}Y[C]=$}}function y(U,L){L=Object.assign({},L);let P={};if(U.hasAttribute("class")){let C=U.getAttribute("class").split(/\s/).filter(Boolean).map(V=>V.trim());for(let V=0;V<C.length;V++)P=Object.assign(P,ee["."+C[V]])}U.hasAttribute("id")&&(P=Object.assign(P,ee["#"+U.getAttribute("id")]));function z(C,V,D){D===void 0&&(D=function($){return $}),U.hasAttribute(C)&&(L[V]=D(U.getAttribute(C))),P[V]&&(L[V]=D(P[V])),U.style&&U.style[C]!==""&&(L[V]=D(U.style[C]))}function he(C){return Math.max(0,Math.min(1,x(C)))}function me(C){return Math.max(0,x(C))}return z("fill","fill"),z("fill-opacity","fillOpacity",he),z("fill-rule","fillRule"),z("opacity","opacity",he),z("stroke","stroke"),z("stroke-opacity","strokeOpacity",he),z("stroke-width","strokeWidth",me),z("stroke-linejoin","strokeLineJoin"),z("stroke-linecap","strokeLineCap"),z("stroke-miterlimit","strokeMiterLimit",me),z("visibility","visibility"),L}function m(U,L){return U-(L-U)}function g(U,L,P){if(typeof U!="string")throw new TypeError("Invalid input: "+typeof U);let z={SEPARATOR:/[ \t\r\n\,.\-+]/,WHITESPACE:/[ \t\r\n]/,DIGIT:/[\d]/,SIGN:/[-+]/,POINT:/\./,COMMA:/,/,EXP:/e/i,FLAGS:/[01]/},he=0,me=1,C=2,V=3,D=he,j=!0,$="",le="",te=[];function ne(Ae,T,_){let O=new SyntaxError('Unexpected character "'+Ae+'" at index '+T+".");throw O.partial=_,O}function k(){$!==""&&(le===""?te.push(Number($)):te.push(Number($)*Math.pow(10,Number(le)))),$="",le=""}let b,we=U.length;for(let Ae=0;Ae<we;Ae++){if(b=U[Ae],Array.isArray(L)&&L.includes(te.length%P)&&z.FLAGS.test(b)){D=me,$=b,k();continue}if(D===he){if(z.WHITESPACE.test(b))continue;if(z.DIGIT.test(b)||z.SIGN.test(b)){D=me,$=b;continue}if(z.POINT.test(b)){D=C,$=b;continue}z.COMMA.test(b)&&(j&&ne(b,Ae,te),j=!0)}if(D===me){if(z.DIGIT.test(b)){$+=b;continue}if(z.POINT.test(b)){$+=b,D=C;continue}if(z.EXP.test(b)){D=V;continue}z.SIGN.test(b)&&$.length===1&&z.SIGN.test($[0])&&ne(b,Ae,te)}if(D===C){if(z.DIGIT.test(b)){$+=b;continue}if(z.EXP.test(b)){D=V;continue}z.POINT.test(b)&&$[$.length-1]==="."&&ne(b,Ae,te)}if(D===V){if(z.DIGIT.test(b)){le+=b;continue}if(z.SIGN.test(b)){if(le===""){le+=b;continue}le.length===1&&z.SIGN.test(le)&&ne(b,Ae,te)}}z.WHITESPACE.test(b)?(k(),D=he,j=!1):z.COMMA.test(b)?(k(),D=he,j=!0):z.SIGN.test(b)?(k(),D=me,$=b):z.POINT.test(b)?(k(),D=C,$=b):ne(b,Ae,te)}return k(),te}let E=["mm","cm","in","pt","pc","px"],S={mm:{mm:1,cm:.1,in:1/25.4,pt:72/25.4,pc:6/25.4,px:-1},cm:{mm:10,cm:1,in:1/2.54,pt:72/2.54,pc:6/2.54,px:-1},in:{mm:25.4,cm:2.54,in:1,pt:72,pc:6,px:-1},pt:{mm:25.4/72,cm:2.54/72,in:1/72,pt:1,pc:6/72,px:-1},pc:{mm:25.4/6,cm:2.54/6,in:1/6,pt:72/6,pc:1,px:-1},px:{px:1}};function x(U){let L="px";if(typeof U=="string"||U instanceof String)for(let z=0,he=E.length;z<he;z++){let me=E[z];if(U.endsWith(me)){L=me,U=U.substring(0,U.length-me.length);break}}let P;return L==="px"&&t.defaultUnit!=="px"?P=S.in[t.defaultUnit]/t.defaultDPI:(P=S[L][t.defaultUnit],P<0&&(P=S[L].in*t.defaultDPI)),P*parseFloat(U)}function A(U){if(!(U.hasAttribute("transform")||U.nodeName==="use"&&(U.hasAttribute("x")||U.hasAttribute("y"))))return null;let L=w(U);return G.length>0&&L.premultiply(G[G.length-1]),We.copy(L),G.push(L),L}function w(U){let L=new Ze;if(U.nodeName==="use"&&(U.hasAttribute("x")||U.hasAttribute("y"))){let P=x(U.getAttribute("x")||0),z=x(U.getAttribute("y")||0);L.makeTranslation(P,z)}return U.hasAttribute("transform")&&I(U.getAttribute("transform"),L),L}function I(U,L){let P=ge,z=U.split(")");for(let he=z.length-1;he>=0;he--){let me=z[he].trim();if(me==="")continue;let C=me.indexOf("("),V=me.length;if(C>0&&C<V){let D=me.slice(0,C),j=g(me.slice(C+1));switch(P.identity(),D){case"translate":if(j.length>=1){let $=j[0],le=0;j.length>=2&&(le=j[1]),P.makeTranslation($,le)}break;case"rotate":if(j.length>=1){let $=0,le=0,te=0;$=j[0]*Math.PI/180,j.length>=3&&(le=j[1],te=j[2]),ye.makeTranslation(-le,-te),Me.makeRotation($),Re.multiplyMatrices(Me,ye),ye.makeTranslation(le,te),P.multiplyMatrices(ye,Re)}break;case"scale":if(j.length>=1){let $=j[0],le=$;j.length>=2&&(le=j[1]),P.makeScale($,le)}break;case"skewX":j.length===1&&P.set(1,Math.tan(j[0]*Math.PI/180),0,0,1,0,0,0,1);break;case"skewY":j.length===1&&P.set(1,0,0,Math.tan(j[0]*Math.PI/180),1,0,0,0,1);break;case"matrix":j.length===6&&P.set(j[0],j[2],j[4],j[1],j[3],j[5],0,0,1);break}L.premultiply(P)}}return L}function v(U,L){function P(C){qe.set(C.x,C.y,1).applyMatrix3(L),C.set(qe.x,qe.y)}function z(C){let V=C.xRadius,D=C.yRadius,j=Math.cos(C.aRotation),$=Math.sin(C.aRotation),le=new B(V*j,V*$,0),te=new B(-D*$,D*j,0),ne=le.applyMatrix3(L),k=te.applyMatrix3(L),b=ge.set(ne.x,k.x,0,ne.y,k.y,0,0,0,1),we=ye.copy(b).invert(),_=Me.copy(we).transpose().multiply(we).elements,O=ie(_[0],_[1],_[4]),W=Math.sqrt(O.rt1),Q=Math.sqrt(O.rt2);if(C.xRadius=1/W,C.yRadius=1/Q,C.aRotation=Math.atan2(O.sn,O.cs),!((C.aEndAngle-C.aStartAngle)%(2*Math.PI)<Number.EPSILON)){let Se=ye.set(W,0,0,0,Q,0,0,0,1),ae=Me.set(O.cs,O.sn,0,-O.sn,O.cs,0,0,0,1),fe=Se.multiply(ae).multiply(b),J=xe=>{let{x:ue,y:ve}=new B(Math.cos(xe),Math.sin(xe),0).applyMatrix3(fe);return Math.atan2(ve,ue)};C.aStartAngle=J(C.aStartAngle),C.aEndAngle=J(C.aEndAngle),R(L)&&(C.aClockwise=!C.aClockwise)}}function he(C){let V=N(L),D=H(L);C.xRadius*=V,C.yRadius*=D;let j=V>Number.EPSILON?Math.atan2(L.elements[1],L.elements[0]):Math.atan2(-L.elements[3],L.elements[4]);C.aRotation+=j,R(L)&&(C.aStartAngle*=-1,C.aEndAngle*=-1,C.aClockwise=!C.aClockwise)}let me=U.subPaths;for(let C=0,V=me.length;C<V;C++){let j=me[C].curves;for(let $=0;$<j.length;$++){let le=j[$];le.isLineCurve?(P(le.v1),P(le.v2)):le.isCubicBezierCurve?(P(le.v0),P(le.v1),P(le.v2),P(le.v3)):le.isQuadraticBezierCurve?(P(le.v0),P(le.v1),P(le.v2)):le.isEllipseCurve&&(ze.set(le.aX,le.aY),P(ze),le.aX=ze.x,le.aY=ze.y,F(L)?z(le):he(le))}}}function R(U){let L=U.elements;return L[0]*L[4]-L[1]*L[3]<0}function F(U){let L=U.elements,P=L[0]*L[3]+L[1]*L[4];if(P===0)return!1;let z=N(U),he=H(U);return Math.abs(P/(z*he))>Number.EPSILON}function N(U){let L=U.elements;return Math.sqrt(L[0]*L[0]+L[1]*L[1])}function H(U){let L=U.elements;return Math.sqrt(L[3]*L[3]+L[4]*L[4])}function oe(U){let L=U.elements,P=L[0]*L[4]-L[1]*L[3];return Math.sqrt(Math.abs(P))}function ie(U,L,P){let z,he,me,C,V,D=U+P,j=U-P,$=Math.sqrt(j*j+4*L*L);return D>0?(z=.5*(D+$),V=1/z,he=U*V*P-L*V*L):D<0?he=.5*(D-$):(z=.5*$,he=-.5*$),j>0?me=j+$:me=j-$,Math.abs(me)>2*Math.abs(L)?(V=-2*L/me,C=1/Math.sqrt(1+V*V),me=V*C):Math.abs(L)===0?(me=1,C=0):(V=-.5*me/L,me=1/Math.sqrt(1+V*V),C=V*me),j>0&&(V=me,me=-C,C=V),{rt1:z,rt2:he,cs:me,sn:C}}let X=[],ee={},Y={},G=[],ge=new Ze,ye=new Ze,Me=new Ze,Re=new Ze,ze=new de,qe=new B,We=new Ze,pe=new DOMParser().parseFromString(e,"image/svg+xml");return p(pe),n(pe.documentElement,{fill:"#000",fillOpacity:1,strokeOpacity:1,strokeWidth:1,strokeLineJoin:"miter",strokeLineCap:"butt",strokeMiterLimit:4}),{paths:X,gradients:Y,xml:pe.documentElement}}static createFillMaterial(e){let t=e.userData.style;if(t.fill===void 0||t.fill==="none")return null;let n=e.color,s=null,r=kd.exec(t.fill);if(r){let a=e.userData.gradients&&e.userData.gradients[r[1]];s=dv(a,e)}let o=new Ot({opacity:t.fillOpacity*(t.opacity||1),transparent:!0,side:Bt,depthWrite:!1});return s!==null?o.map=s:o.color=n,o}static createStrokeMaterial(e){let t=e.userData.style;return t.stroke===void 0||t.stroke==="none"?null:(kd.test(t.stroke)&&console.warn("THREE.SVGLoader: Gradient strokes are not supported."),new Ot({color:new Ge().setStyle(t.stroke,Co),opacity:t.strokeOpacity*(t.opacity||1),transparent:!0,side:Bt,depthWrite:!1}))}static createShapes(e){return console.warn("SVGLoader: createShapes() is deprecated. Use shapePath.toShapes() instead."),e.toShapes()}static getStrokeStyle(e,t,n,s,r){return e=e!==void 0?e:1,t=t!==void 0?t:"#000",n=n!==void 0?n:"miter",s=s!==void 0?s:"butt",r=r!==void 0?r:4,{strokeColor:t,strokeWidth:e,strokeLineJoin:n,strokeLineCap:s,strokeMiterLimit:r}}static pointsToStroke(e,t,n,s){let r=[],o=[],a=[];if(i.pointsToStrokeWithBuffers(e,t,n,s,r,o,a)===0)return null;let c=new wt;return c.setAttribute("position",new Mt(r,3)),c.setAttribute("normal",new Mt(o,3)),c.setAttribute("uv",new Mt(a,2)),c}static pointsToStrokeWithBuffers(e,t,n,s,r,o,a,c){let l=new de,h=new de,u=new de,f=new de,d=new de,p=new de,y=new de,m=new de,g=new de,E=new de,S=new de,x=new de,A=new de,w=new de,I=new de,v=new de,R=new de;n=n!==void 0?n:12,s=s!==void 0?s:.001,c=c!==void 0?c:0,e=me(e);let F=e.length;if(F<2)return 0;let N=e[0].equals(e[F-1]),H,oe=e[0],ie,X=t.strokeWidth/2,ee=1/(F-1),Y=0,G,ge,ye,Me,Re=!1,ze=0,qe=c*3,We=c*2;pe(e[0],e[1],l).multiplyScalar(X),m.copy(e[0]).sub(l),g.copy(e[0]).add(l),E.copy(m),S.copy(g);for(let C=1;C<F;C++){H=e[C],C===F-1?N?ie=e[1]:ie=void 0:ie=e[C+1];let V=l;if(pe(oe,H,V),u.copy(V).multiplyScalar(X),x.copy(H).sub(u),A.copy(H).add(u),G=Y+ee,ge=!1,ie!==void 0){pe(H,ie,h),u.copy(h).multiplyScalar(X),w.copy(H).sub(u),I.copy(H).add(u),ye=!0,u.subVectors(ie,oe),V.dot(u)<0&&(ye=!1),C===1&&(Re=ye),u.subVectors(ie,H),u.normalize();let D=Math.abs(V.dot(u));if(D>Number.EPSILON){let j=X/D;u.multiplyScalar(-j),f.subVectors(H,oe),d.copy(f).setLength(j).add(u),v.copy(d).negate();let $=d.length(),le=f.length();f.divideScalar(le),p.subVectors(ie,H);let te=p.length();if(p.divideScalar(te),f.dot(v)<le&&p.dot(v)<te&&(ge=!0),R.copy(d).add(H),v.add(H),ge){let ne=ye?g:m,k=(R.x-ne.x)*(v.y-ne.y)-(R.y-ne.y)*(v.x-ne.x);(ye&&k<0||!ye&&k>0)&&v.copy(ne)}switch(Me=!1,ge?ye?(I.copy(v),A.copy(v)):(w.copy(v),x.copy(v)):L(),t.strokeLineJoin){case"bevel":P(ye,ge,G);break;case"round":z(ye,ge),ye?U(H,x,w,G,0):U(H,I,A,G,1);break;case"miter":case"miter-clip":default:let ne=X*t.strokeMiterLimit/$;if(ne<1)if(t.strokeLineJoin!=="miter-clip"){P(ye,ge,G);break}else z(ye,ge),ye?(p.subVectors(R,x).multiplyScalar(ne).add(x),y.subVectors(R,w).multiplyScalar(ne).add(w),q(x,G,0),q(p,G,0),q(H,G,.5),q(H,G,.5),q(p,G,0),q(y,G,0),q(H,G,.5),q(y,G,0),q(w,G,0)):(p.subVectors(R,A).multiplyScalar(ne).add(A),y.subVectors(R,I).multiplyScalar(ne).add(I),q(A,G,1),q(p,G,1),q(H,G,.5),q(H,G,.5),q(p,G,1),q(y,G,1),q(H,G,.5),q(y,G,1),q(I,G,1));else ge?(ye?(q(g,Y,1),q(m,Y,0),q(R,G,0),q(g,Y,1),q(R,G,0),q(v,G,1)):(q(g,Y,1),q(m,Y,0),q(R,G,1),q(m,Y,0),q(v,G,0),q(R,G,1)),ye?w.copy(R):I.copy(R)):ye?(q(x,G,0),q(R,G,0),q(H,G,.5),q(H,G,.5),q(R,G,0),q(w,G,0)):(q(A,G,1),q(R,G,1),q(H,G,.5),q(H,G,.5),q(R,G,1),q(I,G,1)),Me=!0;break}}else L()}else L();!N&&C===F-1&&he(e[0],E,S,ye,!0,Y),Y=G,oe=H,m.copy(w),g.copy(I)}if(!N)he(H,x,A,ye,!1,G);else if(ge&&r){let C=R,V=v;Re!==ye&&(C=v,V=R),ye?(Me||Re)&&(V.toArray(r,0),V.toArray(r,9),Me&&C.toArray(r,3)):(Me||!Re)&&(V.toArray(r,3),V.toArray(r,9),Me&&C.toArray(r,0))}if(r){let C=[new de,new de,new de],V=c*3;for(let D=V;D<qe;D+=9)C[0].set(r[D],r[D+1]),C[1].set(r[D+3],r[D+4]),C[2].set(r[D+6],r[D+7]),Kn.area(C)<0&&(r[D+3]=C[0].x,r[D+4]=C[0].y)}return ze;function pe(C,V,D){return D.subVectors(V,C),D.set(-D.y,D.x).normalize()}function q(C,V,D){r&&(r[qe]=C.x,r[qe+1]=C.y,r[qe+2]=0,o&&(o[qe]=0,o[qe+1]=0,o[qe+2]=1),qe+=3,a&&(a[We]=V,a[We+1]=D,We+=2)),ze+=3}function U(C,V,D,j,$){l.copy(V).sub(C).normalize(),h.copy(D).sub(C).normalize();let le=Math.PI,te=l.dot(h);Math.abs(te)<1&&(le=Math.abs(Math.acos(te))),le/=n,u.copy(V);for(let ne=0,k=n-1;ne<k;ne++)f.copy(u).rotateAround(C,le),q(u,j,$),q(f,j,$),q(C,j,.5),u.copy(f);q(u,j,$),q(D,j,$),q(C,j,.5)}function L(){q(g,Y,1),q(m,Y,0),q(x,G,0),q(g,Y,1),q(x,G,0),q(A,G,1)}function P(C,V,D){V?C?(q(g,Y,1),q(m,Y,0),q(x,G,0),q(g,Y,1),q(x,G,0),q(v,G,1),q(x,D,0),q(w,D,0),q(v,D,.5)):(q(g,Y,1),q(m,Y,0),q(A,G,1),q(m,Y,0),q(v,G,0),q(A,G,1),q(A,D,1),q(v,D,0),q(I,D,1)):C?(q(x,D,0),q(w,D,0),q(H,D,.5)):(q(A,D,1),q(I,D,0),q(H,D,.5))}function z(C,V){V&&(C?(q(g,Y,1),q(m,Y,0),q(x,G,0),q(g,Y,1),q(x,G,0),q(v,G,1),q(x,Y,0),q(H,G,.5),q(v,G,1),q(H,G,.5),q(w,Y,0),q(v,G,1)):(q(g,Y,1),q(m,Y,0),q(A,G,1),q(m,Y,0),q(v,G,0),q(A,G,1),q(A,Y,1),q(v,G,0),q(H,G,.5),q(H,G,.5),q(v,G,0),q(I,Y,1)))}function he(C,V,D,j,$,le){switch(t.strokeLineCap){case"round":$?U(C,D,V,le,.5):U(C,V,D,le,.5);break;case"square":if($)l.subVectors(V,C),h.set(l.y,-l.x),u.addVectors(l,h).add(C),f.subVectors(h,l).add(C),j?(u.toArray(r,3),f.toArray(r,0),f.toArray(r,9)):(u.toArray(r,3),a[7]===1?f.toArray(r,9):u.toArray(r,9),f.toArray(r,0));else{l.subVectors(D,C),h.set(l.y,-l.x),u.addVectors(l,h).add(C),f.subVectors(h,l).add(C);let te=r.length;j?(u.toArray(r,te-3),f.toArray(r,te-6),f.toArray(r,te-12)):(f.toArray(r,te-6),u.toArray(r,te-3),f.toArray(r,te-12))}break;case"butt":default:break}}function me(C){let V=!1;for(let j=1,$=C.length-1;j<$;j++)if(C[j].distanceTo(C[j+1])<s){V=!0;break}if(!V)return C;let D=[];D.push(C[0]);for(let j=1,$=C.length-1;j<$;j++)C[j].distanceTo(C[j+1])>=s&&D.push(C[j]);return D.push(C[C.length-1]),D}}},kd=/^\s*url\(\s*(?:["']\s*)?#([^)'"\s]+)(?:\s*["'])?\s*\)\s*$/;function dv(i,e,t=256){if(!i||!Array.isArray(i.stops)||i.stops.length===0)return null;let n=e.userData.transform,s=i.gradientUnits==="objectBoundingBox",r=null;if(s&&(r=pv(e,n),r===null))return null;function o(u,f,d){d.set(u,f,1),i.gradientTransform&&d.applyMatrix3(i.gradientTransform),s&&d.set(r.minX+d.x*r.width,r.minY+d.y*r.height,1),n&&d.applyMatrix3(n)}let a=document.createElement("canvas"),c;if(i.type==="linearGradient"){a.width=t,a.height=1;let u=a.getContext("2d"),f=u.createLinearGradient(0,0,t,0);zd(f,i.stops),u.fillStyle=f,u.fillRect(0,0,t,1);let d=new B,p=new B;o(i.x1,i.y1,d),o(i.x2,i.y2,p);let y=p.x-d.x,m=p.y-d.y,g=y*y+m*m||1e-20,E=y/g,S=m/g,x=-(E*d.x+S*d.y);c=new Ze().set(E,S,x,0,0,.5,0,0,1)}else{let u=i.cx,f=i.cy,d=i.fx,p=i.fy,y=i.r;if(i.gradientTransform){let v=new B;v.set(u,f,1).applyMatrix3(i.gradientTransform),u=v.x,f=v.y,v.set(d,p,1).applyMatrix3(i.gradientTransform),d=v.x,p=v.y}if(s&&(u=r.minX+u*r.width,f=r.minY+f*r.height,d=r.minX+d*r.width,p=r.minY+p*r.height,y=y*Math.sqrt((r.width*r.width+r.height*r.height)/2)),y<=0)return null;a.width=t,a.height=t;let m=a.getContext("2d"),g=u-y,E=f-y,S=2*y,x=t/S;m.setTransform(x,0,0,x,-g*x,-E*x);let A=m.createRadialGradient(d,p,0,u,f,y);zd(A,i.stops),m.fillStyle=A,m.fillRect(g,E,S,S);let w=n?n.clone().invert():new Ze;c=new Ze().set(1/S,0,-g/S,0,1/S,-E/S,0,0,1).multiply(w)}let l=new Qn(a);l.colorSpace=Co,l.flipY=!1,l.matrixAutoUpdate=!1,l.matrix=c;let h=i.spreadMethod==="reflect"?zi:i.spreadMethod==="repeat"?Jn:Gt;return l.wrapS=h,l.wrapT=h,l}function pv(i,e){let t=e?e.clone().invert():null,n=new de,s=new qi;for(let r of i.subPaths)for(let o of r.getPoints())n.copy(o),t&&n.applyMatrix3(t),s.expandByPoint(n);return s.isEmpty()?null:{minX:s.min.x,minY:s.min.y,width:s.max.x-s.min.x,height:s.max.y-s.min.y}}function zd(i,e){let t=new Ge;for(let n of e){let s=n.color;if(n.opacity<1){t.setStyle(n.color,Co);let r=/rgb\(([^)]+)\)/.exec(t.getStyle(Co));r&&(s=`rgba(${r[1]},${n.opacity})`)}i.addColorStop(Math.max(0,Math.min(1,n.offset)),s)}}var mv={src:"",ior:1.75,thickness:4,roughness:.25,dispersion:1.5,clearcoat:.5,tint:"",tintDensity:2,depth:.1,bevel:1,highlight:"#066aff",environmentIntensity:1,background:"",backgroundImage:"",scale:3,xOffset:0,yOffset:0,floatIntensity:1,rotationIntensity:1,floatSpeed:2,orbit:!0,zoom:!1,autoRotate:!1,autoRotateSpeed:2,fov:55,cameraDistance:4,dracoDecoderPath:"https://www.gstatic.com/draco/versioned/decoders/1.5.7/",onLoad:null,onError:null},Hd=new B(0,-1,4).normalize(),su=.3,Vd=30,gv=768,_v=64,xv=[{position:[-10.906,-1,1.846],rotation:[0,-.195,0],scale:[2.328,7.905,4.651]},{position:[-5.607,-.754,-.758],rotation:[0,.994,0],scale:[1.97,1.534,3.955]},{position:[6.167,-.16,7.803],rotation:[0,.561,0],scale:[3.927,6.285,3.687]},{position:[-2.017,.018,6.124],rotation:[0,.333,0],scale:[2.002,4.566,2.064]},{position:[2.291,-.756,-2.621],rotation:[0,-.286,0],scale:[1.546,1.552,1.496]},{position:[-2.193,-.369,-5.547],rotation:[0,.516,0],scale:[3.875,3.487,2.986]}],yv=[{kind:"ring",intensity:15,position:[2,3,-2],scale:[10,10,10],lookAtCenter:!0},{kind:"box",intensity:80,position:[-14,10,8],scale:[.1,2.5,2.5]},{kind:"box",intensity:80,position:[-14,14,-4],scale:[.1,2.5,2.5],withLight:!0},{kind:"box",intensity:23,position:[14,12,0],scale:[.1,5,5],withLight:!0},{kind:"box",intensity:16,position:[0,9,14],scale:[5,5,.1],withLight:!0},{kind:"box",intensity:80,position:[7,8,-14],scale:[2.5,2.5,.1],withLight:!0},{kind:"box",intensity:80,position:[-7,16,-14],scale:[2.5,2.5,.1],withLight:!0},{kind:"box",intensity:1,position:[0,20,0],scale:[.1,.1,.1],withLight:!0},{kind:"box",intensity:20,position:[0,15,0],scale:[10,1,10],withLight:!0}];function vv(i){let e=i.getAttribute("position"),t=i.getAttribute("normal"),n=new B,s=new B,r=new B,o=new B,a=new B;for(let c of i.groups)if(c.materialIndex===0)for(let l=c.start;l<c.start+c.count;l+=3){n.fromBufferAttribute(e,l),s.fromBufferAttribute(e,l+1),r.fromBufferAttribute(e,l+2),o.subVectors(r,s),a.subVectors(n,s),o.cross(a).normalize();for(let h=0;h<3;h++)t.setXYZ(l+h,o.x,o.y,o.z)}t.needsUpdate=!0}function Oc(i,e){i.traverse(t=>{let n=t;n.geometry&&n.geometry.dispose();let s=Array.isArray(n.material)?n.material:[n.material];for(let r of s)if(!(!r||r===e)){for(let o of Object.values(r))o instanceof Tt&&o.dispose();r.dispose()}})}function bv(i){if(i.length<4)return null;let e=(n,s)=>{for(let r=0;r<s.length;r++)if(i[n+r]!==s.charCodeAt(r))return!1;return!0};if(e(0,"glTF"))return"glb";if(i[0]===137&&e(1,"PNG")||i[0]===255&&i[1]===216||e(0,"RIFF")&&e(8,"WEBP")||e(0,"GIF8"))return"bitmap";let t="";try{t=new TextDecoder().decode(i.subarray(0,2048)).replace(/^\uFEFF/,"").trimStart()}catch{return null}return t.startsWith("{")?"gltf":t.startsWith("<")&&t.includes("<svg")?"svg":null}function Mv(i){return new Promise((e,t)=>{let n=URL.createObjectURL(i),s=new Image;s.onload=()=>{URL.revokeObjectURL(n);let r=s.naturalWidth||1024,o=s.naturalHeight||1024,a=Math.min(1,gv/Math.max(r,o)),c=document.createElement("canvas");c.width=Math.max(1,Math.round(r*a)),c.height=Math.max(1,Math.round(o*a));let l=c.getContext("2d");if(!l){t(new Error("2d context unavailable"));return}l.drawImage(s,0,0,c.width,c.height),e(l.getImageData(0,0,c.width,c.height))},s.onerror=()=>{URL.revokeObjectURL(n),t(new Error("Could not decode the image"))},s.src=n})}function Sv(i,e,t){let n=(d,p)=>d>=0&&p>=0&&d<e&&p<t&&i[p*e+d]===1?1:0,s=[],r=(d,p)=>[d-.5,p-1],o=(d,p)=>[d-.5,p],a=(d,p)=>[d-1,p-.5],c=(d,p)=>[d,p-.5];for(let d=0;d<=t;d++)for(let p=0;p<=e;p++)switch(n(p-1,d-1)*8+n(p,d-1)*4+n(p,d)*2+n(p-1,d)){case 1:s.push([a(p,d),o(p,d)]);break;case 2:s.push([o(p,d),c(p,d)]);break;case 3:s.push([a(p,d),c(p,d)]);break;case 4:s.push([r(p,d),c(p,d)]);break;case 5:s.push([a(p,d),r(p,d)]),s.push([o(p,d),c(p,d)]);break;case 6:s.push([r(p,d),o(p,d)]);break;case 7:s.push([a(p,d),r(p,d)]);break;case 8:s.push([a(p,d),r(p,d)]);break;case 9:s.push([r(p,d),o(p,d)]);break;case 10:s.push([r(p,d),c(p,d)]),s.push([a(p,d),o(p,d)]);break;case 11:s.push([r(p,d),c(p,d)]);break;case 12:s.push([a(p,d),c(p,d)]);break;case 13:s.push([o(p,d),c(p,d)]);break;case 14:s.push([a(p,d),o(p,d)]);break}let l=d=>(Math.round(d[0]*2)+4)*8192+Math.round(d[1]*2)+4,h=new Map;for(let d=0;d<s.length;d++)for(let p of s[d]){let y=l(p),m=h.get(y);m?m.push(d):h.set(y,[d])}let u=new Uint8Array(s.length),f=[];for(let d=0;d<s.length;d++){if(u[d])continue;u[d]=1;let p=[s[d][0]],y=s[d][1],m=l(s[d][0]);for(;l(y)!==m;){p.push(y);let g=h.get(l(y))??[],E=-1;for(let A of g)if(!u[A]){E=A;break}if(E<0)break;u[E]=1;let[S,x]=s[E];y=l(S)===l(y)?x:S}p.length>=4&&f.push(p)}return f}function Tv(i,e){if(i.length<6)return i;let t=new Uint8Array(i.length);t[0]=1,t[i.length-1]=1;let n=[[0,i.length-1]];for(;n.length;){let[r,o]=n.pop(),[a,c]=i[r],[l,h]=i[o],u=l-a,f=h-c,d=Math.hypot(u,f)||1e-9,p=-1,y=e;for(let m=r+1;m<o;m++){let g=Math.abs((i[m][0]-a)*f-(i[m][1]-c)*u)/d;g>y&&(y=g,p=m)}p>0&&(t[p]=1,n.push([r,p],[p,o]))}let s=[];for(let r=0;r<i.length;r++)t[r]&&s.push(i[r]);return s}function Ev(i,e){let t=i;for(let n=0;n<e;n++){let s=[];for(let r=0;r<t.length;r++){let[o,a]=t[r],[c,l]=t[(r+1)%t.length];s.push([o*.75+c*.25,a*.75+l*.25],[o*.25+c*.75,a*.25+l*.75])}t=s}return t}function Fc(i){let e=0;for(let t=0;t<i.length;t++){let[n,s]=i[t],[r,o]=i[(t+1)%i.length];e+=n*o-r*s}return e/2}function Gd(i,e){let t=i.length;if(t<3)return i;let n=[];for(let s=0;s<t;s++){let r=i[(s-1+t)%t],o=i[s],a=i[(s+1)%t],c=o.clone().sub(r),l=a.clone().sub(o),h=c.length(),u=l.length();if(h<1e-9||u<1e-9)continue;c.divideScalar(h),l.divideScalar(u);let f=Math.acos(Math.min(Math.max(c.dot(l),-1),1));if(f<.1){n.push(o.clone());continue}let d=Math.min(e,h*.5,u*.5),p=o.clone().addScaledVector(c,-d),y=o.clone().addScaledVector(l,d),m=Math.max(2,Math.ceil(f/.3));for(let g=0;g<=m;g++){let E=g/m,S=(1-E)*(1-E),x=2*(1-E)*E,A=E*E;n.push(new de(S*p.x+x*o.x+A*y.x,S*p.y+x*o.y+A*y.y))}}return n.length>=3?n:i}function Wd(i){return i.length>1&&i[0].distanceToSquared(i[i.length-1])<1e-12?i.slice(0,-1):i}function Av(i,e){return e<1e-6?i:i.map(t=>{let n=t.extractPoints(24),s=new kn(Gd(Wd(n.shape),e));for(let r of n.holes)s.holes.push(new tn(Gd(Wd(r),e)));return s})}function Xd(i,e,t){let n=!1;for(let s=0,r=i.length-1;s<i.length;r=s++){let[o,a]=i[s],[c,l]=i[r];a>t!=l>t&&e<(c-o)*(t-a)/(l-a)+o&&(n=!n)}return n}function wv(i){let{width:e,height:t}=i,n=new Uint8Array(e*t);for(let l=0;l<e*t;l++)n[l]=i.data[l*4+3]>=_v?1:0;let s=Sv(n,e,t),r=[];for(let l of s){let h=Ev(Tv(l,1),2);Math.abs(Fc(h))>12&&r.push(h)}if(r.sort((l,h)=>Math.abs(Fc(h))-Math.abs(Fc(l))),r=r.slice(0,48),r.length===0)throw new Error("No opaque pixels to trace");let o=r.map((l,h)=>{let[u,f]=l[0],d=0;for(let p=0;p<r.length;p++)p!==h&&Xd(r[p],u,f)&&d++;return d}),a=[],c=[];for(let l=0;l<r.length;l++){if(o[l]%2!==0)continue;let h=new kn(r[l].map(([u,f])=>new de(u,f)));a.push(h),c.push({loop:r[l],area:Math.abs(Fc(r[l])),shape:h})}for(let l=0;l<r.length;l++){if(o[l]%2===0)continue;let[h,u]=r[l][0],f=null;for(let d of c)Xd(d.loop,h,u)&&(!f||d.area<f.area)&&(f=d);f?.shape.holes.push(new tn(r[l].map(([d,p])=>new de(d,p))))}return a}function Rv(i){let e=new Sr().parse(i),t=[];for(let n of e.paths)n.userData?.style?.fill!=="none"&&t.push(...Sr.createShapes(n));if(t.length===0)for(let n of e.paths)t.push(...Sr.createShapes(n));if(t.length===0)throw new Error("No fillable shapes in the SVG");return t}function qd(i,e={}){let{canvas:t}=i,n={...mv,...e},s;try{s=new _r({canvas:t,antialias:!0,alpha:!0,powerPreference:"high-performance"})}catch{return null}s.toneMapping=bs;let r=new Bn,o=new Ct(n.fov,1,.1,200);o.position.copy(Hd).multiplyScalar(n.cameraDistance);let a=new Ut;a.position.y=su;let c=new Ut;a.add(c),r.add(a);let l=new yr(o,t);l.enableDamping=!0,l.enablePan=!1,r.add(o);let h=new Ot,u=new ht(new xs(1,1),h);u.position.set(0,0,-Vd),u.visible=!1,o.add(u);let f=new vs;f.setCrossOrigin("anonymous");let d=null,p=null;function y(){if(!d)return;let te=2*Vd*Math.tan(Ts.degToRad(o.fov)/2),ne=te*o.aspect;u.scale.set(ne,te,1);let k=d.image,b=ne/te,we=k.width/k.height;we>b?(d.repeat.set(b/we,1),d.offset.set((1-d.repeat.x)/2,0)):(d.repeat.set(1,we/b),d.offset.set(0,(1-d.repeat.y)/2))}function m(){let te=n.backgroundImage;if(te!==p){if(p=te,!te){u.visible=!1,h.map=null,d?.dispose(),d=null,w=!0;return}f.load(te,ne=>{if(ee||n.backgroundImage!==te){ne.dispose();return}ne.colorSpace=at,ne.anisotropy=s.capabilities.getMaxAnisotropy(),ne.generateMipmaps=!1,ne.minFilter=bt,d?.dispose(),d=ne,h.map=ne,h.needsUpdate=!0,u.visible=!0,y(),w=!0})}}let g=new $t({color:16777215,metalness:0,transmission:1,clearcoatRoughness:.06,specularIntensity:1}),E=new Qi(s),S=null,x=null,A=null,w=!0;function I(){S=new Bn;let te=new Ut;te.position.set(0,-.5,0),S.add(te);for(let[Ae,T]of[[-15,15],[15,15],[15,-15],[-15,-15]]){let _=new Si(16777215,2,0,.2,1,0);_.position.set(Ae,20,T),te.add(_,_.target)}let ne=new Hn(16777215,100,28,2);ne.position.set(.5,14,.5),te.add(ne);let k=new En,b=new ht(k,new an({color:"gray",side:Ft}));b.position.set(0,13.2,0),b.scale.set(31.5,28.5,31.5),te.add(b);let we=new an({color:16777215});for(let Ae of xv){let T=new ht(k,we);T.position.set(...Ae.position),T.rotation.set(...Ae.rotation),T.scale.set(...Ae.scale),te.add(T)}for(let Ae of yv){let T=Ae.kind==="ring"?new ys(.5,1,64):new En,_=new Ot({side:Bt,toneMapped:!1});_.color.set(Ae.kind==="ring"?n.highlight:"#ffffff").multiplyScalar(Ae.intensity),Ae.kind==="ring"&&(x=_);let O=new ht(T,_);if(O.position.set(...Ae.position),O.scale.set(...Ae.scale),Ae.lookAtCenter&&O.lookAt(0,0,0),te.add(O),Ae.withLight){let W=new Hn(16777215,100,28,2);W.position.set(...Ae.position),te.add(W)}}}function v(){if(d){let te=d.image,ne=document.createElement("canvas");ne.width=64,ne.height=32;let k=ne.getContext("2d");k&&(k.filter="blur(4px)",k.drawImage(te,-4,-4,ne.width+8,ne.height+8));let b=new Qn(ne);b.colorSpace=at,b.mapping=cr,A?.dispose(),A=E.fromEquirectangular(b),b.dispose(),r.environment=A.texture;return}S||I(),x&&x.color.set(n.highlight).multiplyScalar(15),A?.dispose(),A=E.fromScene(S,.6,.1,1e3),r.environment=A.texture}let R=null,F=1,N=null,H=-1,oe=-1,ie=null,X=0,ee=!1,Y=new Mr,G=new vr;G.setDecoderPath(n.dracoDecoderPath),Y.setDRACOLoader(G);function ge(){R&&(c.scale.setScalar(n.scale/F),g.thickness=Math.max(n.thickness,0)/c.scale.x)}function ye(){R&&(c.remove(R),Oc(R,g),R=null)}function Me(){N?.kind==="mesh"&&Oc(N.scene,g),N=null,H=-1,oe=-1,ye()}function Re(te){ye(),R=te;let ne=new Xt().setFromObject(R),k=ne.getSize(new B),b=ne.getCenter(new B);F=Math.max(k.x,k.y,k.z,1e-4),R.position.sub(b),ge(),c.add(R)}function ze(){if(!N)return;if(N.kind==="mesh"){if(R)return;N.scene.traverse(O=>{let W=O;if(!W.isMesh)return;let Q=Array.isArray(W.material)?W.material:[W.material];for(let _e of Q)if(!(!_e||_e===g)){for(let Se of Object.values(_e))Se instanceof Tt&&Se.dispose();_e.dispose()}W.material=g,W.geometry.getAttribute("normal")||W.geometry.computeVertexNormals()}),Re(N.scene);return}let te=Math.min(Math.max(n.depth,.02),1),ne=Math.min(Math.max(n.bevel,0),1);if(R&&te===H&&ne===oe)return;H=te,oe=ne;let k=new qi;for(let O of N.shapes)for(let W of O.getPoints(4))k.expandByPoint(W);let b=Math.max(k.max.x-k.min.x,k.max.y-k.min.y,1e-4),we=te*b,Ae=ne*we*.5,T=Av(N.shapes,Ae*1.25),_=new _s(T,{depth:Math.max(we-Ae*2,we*.1),bevelEnabled:Ae>1e-4,bevelThickness:Ae,bevelSize:Ae*.9,bevelOffset:0,bevelSegments:12,curveSegments:24});_=bd(_,Math.PI/7),vv(_),_.rotateX(Math.PI),Re(new ht(_,g))}async function qe(){let te=n.src;if(te===ie)return;ie=te;let ne=++X;if(!te){Me();return}try{let k=await fetch(te);if(!k.ok)throw new Error(`HTTP ${k.status}`);let b=await k.arrayBuffer();if(ee||ne!==X)return;let we=new Uint8Array(b),Ae=bv(we);if(!Ae)throw new Error("Unrecognized asset format");if(Ae==="glb"||Ae==="gltf"){G.setDecoderPath(n.dracoDecoderPath);let T=te.slice(0,te.lastIndexOf("/")+1),_=Ae==="glb"?b:new TextDecoder().decode(we),O=await Y.parseAsync(_,T);if(ee||ne!==X){Oc(O.scene);return}Me(),N={kind:"mesh",scene:O.scene}}else if(Ae==="svg"){let T=Rv(new TextDecoder().decode(we));if(ee||ne!==X)return;Me(),N={kind:"shapes",shapes:T}}else{let T=await Mv(new Blob([b]));if(ee||ne!==X)return;let _=wv(T);Me(),N={kind:"shapes",shapes:_}}ze(),n.onLoad?.()}catch(k){if(ee||ne!==X)return;n.onError?.(k)}}let We=window.matchMedia("(prefers-reduced-motion: reduce)"),pe=We.matches,q=()=>{pe=We.matches,pe&&a.rotation.set(0,0,0),L()};We.addEventListener("change",q);let U=new Ge;function L(){n.background?(U.set(n.background),r.background=U,s.setClearColor(U,1)):(r.background=null,s.setClearColor(0,0)),r.environmentIntensity=n.environmentIntensity,l.enableRotate=n.orbit,l.enableZoom=n.zoom,l.autoRotate=n.autoRotate&&!pe,l.autoRotateSpeed=n.autoRotateSpeed,o.fov=n.fov,o.updateProjectionMatrix(),m(),y(),a.position.x=n.xOffset,a.position.y=su+n.yOffset,g.ior=Math.min(Math.max(n.ior,1),2.333),g.roughness=Math.min(Math.max(n.roughness,0),1),g.dispersion=Math.max(n.dispersion,0),g.clearcoat=Math.min(Math.max(n.clearcoat,0),1),n.tint?(g.attenuationColor.set(n.tint),g.attenuationDistance=1.5/Math.max(n.tintDensity,.01)):(g.attenuationColor.set(16777215),g.attenuationDistance=1/0),ge(),ze()}function P(){let te=Math.max(t.clientWidth,1),ne=Math.max(t.clientHeight,1),k=Math.min(window.devicePixelRatio||1,2);s.setPixelRatio(k),s.setSize(te,ne,!1),o.aspect=te/ne,o.updateProjectionMatrix(),y()}let z=new ResizeObserver(P);z.observe(t),P(),L(),qe();let he=!0,me=!1;function C(te){if(!he){$=0,D();return}let ne=$?Math.min((te-$)/1e3,.1):0;$=te,w&&(w=!1,v()),l.update(),pe||(le+=ne*n.floatSpeed,a.rotation.x=Math.cos(le/4)/8*n.rotationIntensity,a.rotation.y=Math.sin(le/4)/8*n.rotationIntensity,a.rotation.z=Math.sin(le/4)/20*n.rotationIntensity,a.position.y=su+n.yOffset+Math.sin(le/1.5)/10*n.floatIntensity),s.render(r,o)}function V(){me||!he||ee||(me=!0,s.setAnimationLoop(C))}function D(){me&&(me=!1,s.setAnimationLoop(null))}let j=typeof IntersectionObserver<"u"?new IntersectionObserver(te=>{he=te[te.length-1]?.isIntersecting??!0,he?V():D()}):null;j?.observe(t);let $=0,le=Math.random()*100;return V(),{camera:o,controls:l,setOptions(te){let ne=!1;for(let[we,Ae]of Object.entries(te))if(typeof Ae!="function"&&n[we]!==Ae){ne=!0;break}if(!ne){Object.assign(n,te);return}let k=n.highlight,b=n.cameraDistance;Object.assign(n,te),n.highlight!==k&&(w=!0),n.cameraDistance!==b&&o.position.copy(Hd).multiplyScalar(n.cameraDistance),L(),qe(),V()},resize:P,destroy(){ee=!0,X+=1,D(),z.disconnect(),j?.disconnect(),We.removeEventListener("change",q),l.dispose(),Me(),u.geometry.dispose(),h.dispose(),d?.dispose(),S&&Oc(S),A?.dispose(),E.dispose(),G.dispose(),g.dispose(),s.dispose()}}}var Cv={zone:240,angle:80,rounding:150,perspective:700,direction:"in",ease:240,smoothing:.1,top:!0,bottom:!0,tumble:.5,tilt:.5},Pv=`#version 300 es
precision highp float;
layout(location = 0) in vec2 aPos;
out vec2 vUv;
void main () {
  vUv = aPos * 0.5 + 0.5;
  gl_Position = vec4(aPos, 0.0, 1.0);
}`,Iv=`#version 300 es
precision highp float;
in vec2 vUv;
out vec4 outColor;
uniform sampler2D uContent;
uniform float uZone;
uniform float uAngle;
uniform float uPersp;
uniform float uDir;
uniform float uTopAmt;
uniform float uBotAmt;
uniform float uMaxX;
uniform float uPxY;
uniform float uPxX;
uniform float uCover;
uniform vec3 uBg;
uniform float uTiltX;
uniform float uTiltY;
uniform float uPhi;
uniform float uRound;

vec3 foldEdge (float sy, float amt) {
  float yf = 1.0 - uZone;
  if (amt < 1e-4) return vec3(sy, 0.0, 1.0);
  float theta = uAngle * amt;
  if (uRound < 1e-4) {
    float s = sin(theta) * uDir;
    float c = cos(theta);
    float denom = max(c * uPersp + s * (0.5 - sy), 1e-5);
    float tRaw = uPersp * (sy - yf) / denom;
    float t = clamp(tRaw, 0.0, uZone);
    float z = max(t * s, -0.85 * uPersp);
    float alpha = 1.0 - smoothstep(uZone, uZone + 2.0 * uPxY, tRaw);
    return vec3(yf + t, z, alpha);
  }
  if (sy <= yf) return vec3(sy, 0.0, 1.0);
  float R = min(uRound, uZone);
  float r = R / theta;
  float ca = cos(theta);
  float sa = sin(theta);
  float yA = r * sa;
  float zA = r * (1.0 - ca);
  float prevSy = yf;
  float prevZ = 0.0;
  float prevU = 0.0;
  float bestU = -1.0;
  float bestZ = 0.0;
  float maxSy = yf;
  float du = uZone / 40.0;
  for (int i = 1; i <= 40; i++) {
    float u = du * float(i);
    float Y;
    float Zm;
    if (u <= R) {
      float a = u / r;
      Y = r * sin(a);
      Zm = r * (1.0 - cos(a));
    } else {
      Y = yA + (u - R) * ca;
      Zm = zA + (u - R) * sa;
    }
    Y += yf;
    float Z = max(Zm * uDir, -0.85 * uPersp);
    float scr = 0.5 + (Y - 0.5) * uPersp / (uPersp + Z);
    if ((prevSy - sy) * (scr - sy) <= 0.0 && abs(scr - prevSy) > 1e-7) {
      float f = clamp((sy - prevSy) / (scr - prevSy), 0.0, 1.0);
      bestU = mix(prevU, u, f);
      bestZ = mix(prevZ, Z, f);
      if (uDir > 0.0) break;
    }
    maxSy = max(maxSy, scr);
    prevSy = scr;
    prevZ = Z;
    prevU = u;
  }
  if (bestU < 0.0) {
    float alpha = 1.0 - smoothstep(maxSy - uPxY, maxSy + uPxY, sy);
    return vec3(1.0, prevZ, alpha);
  }
  return vec3(yf + bestU, bestZ, 1.0);
}

vec2 tipPlane (float sy, float phi) {
  float s = sin(phi);
  float c = cos(phi);
  float denom = max(c * uPersp + s * (sy - 0.5), 1e-4);
  float t = uPersp * (1.0 - sy) / denom;
  return vec2(1.0 - t, t * s);
}

void main () {
  vec2 uv = vUv;
  float cx = uMaxX * 0.5;
  float zSum = 0.0;

  if (abs(uPhi) > 1e-4) {
    if (uPhi > 0.0) {
      vec2 r = tipPlane(uv.y, uPhi);
      uv.y = r.x;
      zSum += r.y;
    } else {
      vec2 r = tipPlane(1.0 - uv.y, -uPhi);
      uv.y = 1.0 - r.x;
      zSum += r.y;
    }
  }

  float zG = uTiltX * (uv.x - cx) + uTiltY * (uv.y - 0.5);
  zSum += zG;
  uv.y = 0.5 + (uv.y - 0.5) * (uPersp + zG) / uPersp;

  float inTop = step(1.0 - uZone, uv.y);
  float inBot = step(uv.y, uZone);

  vec3 top = foldEdge(uv.y, uTopAmt);
  vec3 bot = foldEdge(1.0 - uv.y, uBotAmt);

  float srcY = uv.y;
  srcY = mix(srcY, top.x, inTop);
  srcY = mix(srcY, 1.0 - bot.x, inBot);

  zSum += inTop * top.y + inBot * bot.y;
  float alpha = mix(1.0, top.z, inTop) * mix(1.0, bot.z, inBot);

  float srcX = cx + (uv.x - cx) * (uPersp + zSum) / uPersp;

  alpha *= smoothstep(-2.0 * uPxX, 0.0, srcX);
  alpha *= 1.0 - smoothstep(uMaxX, uMaxX + 2.0 * uPxX, srcX);
  alpha *= smoothstep(-2.0 * uPxY, 0.0, srcY);
  alpha *= 1.0 - smoothstep(1.0, 1.0 + 2.0 * uPxY, srcY);

  vec2 p = vec2(
    clamp(srcX, 0.0005, uMaxX - 0.0005),
    clamp(srcY, 0.0005, 0.9995)
  );
  vec4 base = texture(uContent, vec2(p.x, 1.0 - p.y));

  outColor = vec4(mix(uBg, base.rgb, alpha * base.a), uCover);
}`;function Yd(){if(typeof document>"u")return!1;let i=document.createElement("canvas"),e=i.getContext("2d");return!!(e&&typeof e.drawElementImage=="function"&&typeof i.requestPaint=="function")}var ru="data-canvasui-hover",Tr="data-canvasui-content",Lv=`:is([${ru}], :hover:where(:not([${Tr}], [${Tr}] *)))`;function Dv(){if(typeof document>"u"||document.documentElement.dataset.canvasuiHoverRules==="")return;document.documentElement.dataset.canvasuiHoverRules="";let i=t=>{for(let n of Array.from(t))if(n instanceof CSSStyleRule){if(n.selectorText.includes(":hover"))try{n.selectorText=n.selectorText.replace(/:hover\b/g,Lv)}catch{}n.cssRules.length&&i(n.cssRules)}else if("cssRules"in n)try{i(n.cssRules)}catch{}};for(let t of Array.from(document.styleSheets))try{i(t.cssRules)}catch{}let e=document.createElement("style");e.textContent=`[${Tr}], [${Tr}] * { cursor: var(--canvasui-cursor, auto) !important; }`,document.head.appendChild(e)}function Zd(i,e={}){let t={...Cv,...e},{source:n,content:s,output:r}=i,o=r.getContext("webgl2",{alpha:!0,depth:!1,stencil:!1,antialias:!1,premultipliedAlpha:!1});if(!o||o.isContextLost())return null;let a=n.getContext("2d"),c=n,l=!!(a&&typeof a.drawElementImage=="function"&&typeof c.requestPaint=="function"),h=!1,u=()=>{};l&&(c.onpaint=()=>{try{a.reset(),a.drawElementImage(s,0,0),h=!0,u()}catch{}});function f(J,xe){let ue=o.createShader(J);return o.shaderSource(ue,xe),o.compileShader(ue),o.getShaderParameter(ue,o.COMPILE_STATUS)||console.error("Bend shader error:",o.getShaderInfoLog(ue)),ue}let d=f(o.VERTEX_SHADER,Pv),p=f(o.FRAGMENT_SHADER,Iv),y=o.createProgram();o.attachShader(y,d),o.attachShader(y,p),o.linkProgram(y);let m={},g=o.getProgramParameter(y,o.ACTIVE_UNIFORMS);for(let J=0;J<g;J++){let xe=o.getActiveUniform(y,J);m[xe.name]=o.getUniformLocation(y,xe.name)}let E=o.createBuffer();o.bindBuffer(o.ARRAY_BUFFER,E),o.bufferData(o.ARRAY_BUFFER,new Float32Array([-1,-1,1,-1,-1,1,1,1]),o.STATIC_DRAW),o.enableVertexAttribArray(0),o.vertexAttribPointer(0,2,o.FLOAT,!1,0,0);let S=o.createTexture();o.bindTexture(o.TEXTURE_2D,S),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_MIN_FILTER,o.LINEAR_MIPMAP_LINEAR),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_MAG_FILTER,o.LINEAR),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_WRAP_S,o.CLAMP_TO_EDGE),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_WRAP_T,o.CLAMP_TO_EDGE),o.texImage2D(o.TEXTURE_2D,0,o.RGBA,1,1,0,o.RGBA,o.UNSIGNED_BYTE,new Uint8Array([0,0,0,0])),o.generateMipmap(o.TEXTURE_2D);let x=1,A=[0,0,0],w=document.createElement("canvas");w.width=w.height=1;let I=w.getContext("2d",{willReadFrequently:!0});function v(){if(!I)return;let J=s;for(;J;){let xe=getComputedStyle(J).backgroundColor;if(xe&&xe!=="transparent"){I.clearRect(0,0,1,1),I.fillStyle=xe,I.fillRect(0,0,1,1);let[ue,ve,Ee,De]=I.getImageData(0,0,1,1).data;if(De>0){A=[ue/255,ve/255,Ee/255];return}}J=J.parentElement}A=[0,0,0]}function R(){let J=Math.min(window.devicePixelRatio||1,2),xe=Math.max(1,Math.round(r.clientWidth*J)),ue=Math.max(1,Math.round(r.clientHeight*J));if((r.width!==xe||r.height!==ue)&&(r.width=xe,r.height=ue),x=Math.min(1,Math.max(.05,s.clientWidth/Math.max(r.clientWidth,1))),l){let ve=Math.max(1,Math.round(n.clientWidth)),Ee=Math.max(1,Math.round(n.clientHeight));(n.width!==ve*J||n.height!==Ee*J)&&(n.width=ve*J,n.height=Ee*J),c.requestPaint()}}let F=0,N=0,H=0,oe=0,ie=0,X=0,ee=0,Y=0,G=0,ge=0;function ye(){let J=s.scrollHeight-s.clientHeight,xe=s.scrollTop,ue=Math.max(t.ease,1),ve=Ee=>{let De=Math.min(Math.max(Ee/ue,0),1);return De*De*(3-2*De)};F=J>1&&t.top?ve(xe):0,N=J>1&&t.bottom?ve(J-xe):0}R(),ye(),v();function Me(){!l||!h||(h=!1,v(),o.bindTexture(o.TEXTURE_2D,S),o.texImage2D(o.TEXTURE_2D,0,o.RGBA,o.RGBA,o.UNSIGNED_BYTE,n),o.generateMipmap(o.TEXTURE_2D))}function Re(){Me();let J=Math.max(r.clientHeight,1),xe=Math.max(r.clientWidth,1),ue=Math.min(Math.max(t.zone,8)/J,.49);o.useProgram(y),o.activeTexture(o.TEXTURE0),o.bindTexture(o.TEXTURE_2D,S),o.uniform1i(m.uContent,0),o.uniform1f(m.uZone,ue),o.uniform1f(m.uAngle,Math.min(Math.max(t.angle,1),160)*(Math.PI/180)),o.uniform1f(m.uPersp,Math.max(t.perspective,50)/J),o.uniform1f(m.uDir,t.direction==="in"?-1:1),o.uniform1f(m.uTopAmt,H),o.uniform1f(m.uBotAmt,oe),o.uniform1f(m.uMaxX,x),o.uniform1f(m.uPxY,1.5/J),o.uniform1f(m.uPxX,1.5/xe),o.uniform1f(m.uCover,l?1:0),o.uniform3f(m.uBg,A[0],A[1],A[2]),o.uniform1f(m.uTiltX,G),o.uniform1f(m.uTiltY,ge),o.uniform1f(m.uPhi,X),o.uniform1f(m.uRound,Math.min(Math.max(t.rounding,0)/J,ue)),o.bindFramebuffer(o.FRAMEBUFFER,null),o.viewport(0,0,r.width,r.height),o.drawArrays(o.TRIANGLE_STRIP,0,4)}let ze=0,qe=performance.now(),We=!1,pe=!1,q=!0,U=window.matchMedia("(prefers-reduced-motion: reduce)"),L=U.matches;function P(J){if(We)return;if(!q){pe=!1;return}let xe=Math.min((J-qe)/1e3,1/30);qe=J;let ue=t.smoothing,ve=L||ue<=0?1:1-Math.exp(-xe/Math.max(ue,1e-4));H+=(F-H)*ve,oe+=(N-oe)*ve,Math.abs(F-H)<.001&&(H=F),Math.abs(N-oe)<.001&&(oe=N),ie*=Math.exp(-xe/.22),Math.abs(ie)<.5&&(ie=0);let Ee=L||t.tumble<=0?0:Math.tanh(ie/500)*.4*Math.min(t.tumble,1);X+=(Ee-X)*Math.min(xe/.09,1),Ee===0&&Math.abs(X)<1e-4&&(X=0),(L||t.tilt<=0)&&(ee=0,Y=0);let De=Math.min(xe/.15,1);if(G+=(ee-G)*De,ge+=(Y-ge)*De,Math.abs(ee-G)<1e-4&&(G=ee),Math.abs(Y-ge)<1e-4&&(ge=Y),Re(),!h&&H===F&&oe===N&&ie===0&&X===0&&G===ee&&ge===Y){pe=!1;return}ze=requestAnimationFrame(P)}function z(){We||pe||!q||(pe=!0,qe=performance.now(),ze=requestAnimationFrame(P))}u=z,z();function he(){ye(),l&&c.requestPaint(),k&&we(te,ne),z()}s.addEventListener("scroll",he,{passive:!0});function me(J){if(t.tumble<=0||L)return;let xe=s.scrollHeight-s.clientHeight;if(xe<=1)return;let ue=s.scrollTop;if(J.deltaY>0&&ue>=xe-1)ie=Math.min(ie+J.deltaY,900);else if(J.deltaY<0&&ue<=1)ie=Math.max(ie+J.deltaY,-900);else return;z()}s.addEventListener("wheel",me,{passive:!0});function C(J){if(J.isPrimary&&(te=J.clientX,ne=J.clientY,k=!0,we(J.clientX,J.clientY),t.tilt>0&&!L)){let xe=r.getBoundingClientRect();if(xe.width>0&&xe.height>0){let ue=(J.clientX-xe.left)/xe.width-.5,ve=.5-(J.clientY-xe.top)/xe.height,Ee=Math.min(t.tilt,1)*.14;ee=-ue*Ee,Y=-ve*Ee,z()}}}s.addEventListener("pointermove",C,{passive:!0});function V(){k=!1,b(null),ee=0,Y=0,z()}s.addEventListener("pointerleave",V);function D(J,xe){let ue=Math.max(r.clientWidth,1),ve=Math.max(r.clientHeight,1),Ee=Math.max(t.perspective,50)/ve,De=Math.min(Math.max(t.zone,8)/ve,.49),je=Math.min(Math.max(t.rounding,0)/ve,De),Z=Math.min(Math.max(t.angle,1),160)*(Math.PI/180),Pe=t.direction==="in"?-1:1,be=x*.5,Ie=J/ue,Ce=1-xe/ve,Te=0;if(Math.abs(X)>1e-4){let ct=(fn,Cn)=>{let Pn=Math.sin(Cn),Xn=Math.cos(Cn),oi=Math.max(Xn*Ee+Pn*(fn-.5),1e-4),Ei=Ee*(1-fn)/oi;return[1-Ei,Ei*Pn]};if(X>0){let fn=ct(Ce,X);Ce=fn[0],Te+=fn[1]}else{let fn=ct(1-Ce,-X);Ce=1-fn[0],Te+=fn[1]}}let He=G*(Ie-be)+ge*(Ce-.5);Te+=He,Ce=.5+(Ce-.5)*((Ee+He)/Ee);let Be=(ct,fn)=>{let Cn=1-De;if(fn<1e-4)return[ct,0,1];let Pn=Z*fn;if(je<1e-4){let ci=Math.sin(Pn)*Pe,qn=Math.cos(Pn),is=Math.max(qn*Ee+ci*(.5-ct),1e-5),Ri=Ee*(ct-Cn)/is,Ps=Math.min(Math.max(Ri,0),De),M=Math.max(Ps*ci,-.85*Ee);return[Cn+Ps,M,Ri>De?0:1]}if(ct<=Cn)return[ct,0,1];let Xn=Math.min(je,De),oi=Xn/Pn,Ei=Math.cos(Pn),Er=Math.sin(Pn),ai=oi*Er,Ar=oi*(1-Ei),Ai=Cn,ts=0,wi=0,Cs=-1,ns=0,Po=De/40;for(let ci=1;ci<=40;ci++){let qn=Po*ci,is,Ri;if(qn<=Xn){let ce=qn/oi;is=oi*Math.sin(ce),Ri=oi*(1-Math.cos(ce))}else is=ai+(qn-Xn)*Ei,Ri=Ar+(qn-Xn)*Er;let Ps=Cn+is,M=Math.max(Ri*Pe,-.85*Ee),K=.5+(Ps-.5)*Ee/(Ee+M);if((Ai-ct)*(K-ct)<=0&&Math.abs(K-Ai)>1e-7){let ce=Math.min(Math.max((ct-Ai)/(K-Ai),0),1);if(Cs=wi+(qn-wi)*ce,ns=ts+(M-ts)*ce,Pe>0)break}Ai=K,ts=M,wi=qn}return Cs<0?[1,ts,0]:[Cn+Cs,ns,1]},dt=Ce,ot=1;if(Ce>=1-De){let ct=Be(Ce,H);dt=ct[0],Te+=ct[1],ot*=ct[2]}else if(Ce<=De){let ct=Be(1-Ce,oe);dt=1-ct[0],Te+=ct[1],ot*=ct[2]}let sn=be+(Ie-be)*((Ee+Te)/Ee);return(sn<0||sn>x||dt<0||dt>1)&&(ot=0),{x:sn*ue,y:(1-dt)*ve,alpha:ot}}let j=!1,$=[],le=null,te=0,ne=0,k=!1;l&&(Dv(),s.setAttribute(Tr,""));function b(J){if(J===le)return;le=J;let xe=new Set;for(let ue=J;ue&&(xe.add(ue),ue!==s);ue=ue.parentElement);for(let ue of $)xe.has(ue)||ue.removeAttribute(ru);for(let ue of xe)ue.setAttribute(ru,"");$=Array.from(xe),J?s.style.setProperty("--canvasui-cursor",getComputedStyle(J).cursor):s.style.removeProperty("--canvasui-cursor")}function we(J,xe){if(!l)return;let ue=r.getBoundingClientRect();if(ue.width===0||ue.height===0)return;let ve=D(J-ue.left,xe-ue.top);if(ve.alpha<.5){b(null);return}let Ee=document.elementFromPoint(ue.left+ve.x,ue.top+ve.y);b(Ee&&s.contains(Ee)?Ee:null)}function Ae(J){if(j||!l||J.button!==0)return;let xe=r.getBoundingClientRect();if(xe.width===0||xe.height===0)return;let ue=J.clientX-xe.left,ve=J.clientY-xe.top,Ee=D(ue,ve);if(Ee.alpha<.5){J.preventDefault(),J.stopPropagation();return}if(Math.hypot(Ee.x-ue,Ee.y-ve)<1.5)return;J.preventDefault(),J.stopPropagation();let De=document.elementFromPoint(xe.left+Ee.x,xe.top+Ee.y);if(De){j=!0;try{De.dispatchEvent(new MouseEvent("click",{bubbles:!0,cancelable:!0,composed:!0,view:window,detail:J.detail,clientX:xe.left+Ee.x,clientY:xe.top+Ee.y,screenX:J.screenX,screenY:J.screenY,ctrlKey:J.ctrlKey,shiftKey:J.shiftKey,altKey:J.altKey,metaKey:J.metaKey,button:J.button})),De instanceof HTMLElement&&De.matches("input, textarea, select, [contenteditable]")&&De.focus()}finally{j=!1}}}s.addEventListener("click",Ae,!0);function T(J,xe){let ue=document;if(typeof ue.caretPositionFromPoint=="function"){let Ee=ue.caretPositionFromPoint(J,xe);return Ee?{node:Ee.offsetNode,offset:Ee.offset}:null}let ve=ue.caretRangeFromPoint?.(J,xe);return ve?{node:ve.startContainer,offset:ve.startOffset}:null}function _(J){let xe=r.getBoundingClientRect();if(xe.width===0||xe.height===0)return null;let ue=D(J.clientX-xe.left,J.clientY-xe.top);if(ue.alpha<.5)return null;let ve=xe.left+ue.x,Ee=xe.top+ue.y;return Math.hypot(ve-J.clientX,Ee-J.clientY)<1.5?null:{x:ve,y:Ee}}let O=!1;function W(J){if(j||!l||J.button!==0)return;let xe=_(J);if(!xe)return;J.preventDefault();let ue=T(xe.x,xe.y);if(!ue||!s.contains(ue.node))return;let ve=window.getSelection();ve&&(ve.removeAllRanges(),ve.collapse(ue.node,ue.offset),O=!0)}function Q(J){if(!O)return;if(!(J.buttons&1)){O=!1;return}let xe=_(J),ue=xe?T(xe.x,xe.y):null,ve=window.getSelection();ue&&ve&&ve.anchorNode&&s.contains(ue.node)&&ve.extend(ue.node,ue.offset)}function _e(){O=!1}s.addEventListener("mousedown",W,!0),window.addEventListener("mousemove",Q,!0),window.addEventListener("mouseup",_e,!0);function Se(){L=U.matches,z()}U.addEventListener("change",Se);let ae=new ResizeObserver(()=>{R(),ye(),z()});ae.observe(r),ae.observe(s);let fe=new IntersectionObserver(J=>{q=J[J.length-1]?.isIntersecting??!0,q&&z()});return fe.observe(r),{setOptions(J){Object.assign(t,J),ye(),z()},resize(){R(),ye(),z()},destroy(){We=!0,cancelAnimationFrame(ze),b(null),s.removeAttribute(Tr),s.removeEventListener("scroll",he),s.removeEventListener("wheel",me),s.removeEventListener("pointermove",C),s.removeEventListener("pointerleave",V),s.removeEventListener("click",Ae,!0),s.removeEventListener("mousedown",W,!0),window.removeEventListener("mousemove",Q,!0),window.removeEventListener("mouseup",_e,!0),ae.disconnect(),fe.disconnect(),U.removeEventListener("change",Se),o.deleteTexture(S),o.deleteProgram(y),o.deleteShader(d),o.deleteShader(p),o.deleteBuffer(E),l&&(c.onpaint=null)}}}var Nv={zone:240,angle:80,rounding:150,perspective:700,ease:300,smoothing:.5,tumble:.5,tilt:0,direction:"in",top:!1,bottom:!1};function Kd(){let i=document.querySelector("[data-bend]"),e=i?.querySelector(".bend-source"),t=i?.querySelector(".bend-content"),n=i?.querySelector(".bend-output");if(!i||!e||!t||!n||!Yd())return;let s=t.scrollTop;e.hidden=!1,e.appendChild(t);let r=Zd({source:e,content:t,output:n},Nv);return r||(e.hidden=!0,i.insertBefore(t,n)),t.scrollTop=s,r}var $d=800,Uv=14,Rs=(i,e,t)=>i+(e-i)*t,jd=i=>i<0?0:i>1?1:i,Bc=i=>1-Math.pow(1-i,3);function st(i,e,t){let n=i.dataset[e];if(n===void 0)return t;let s=Number(n);return Number.isFinite(s)?s:t}function kc(i,e,t){return i.dataset[e]||t}function Ov(){try{let i=document.createElement("canvas");return!!(i.getContext("webgl2")||i.getContext("webgl"))}catch{return!1}}var Ti={bevel:.02,depth:.175,thickness:1.6,dispersion:1.2,roughness:.03,tint:"#e7c574",tintDensity:.7,highlight:"#066aff"},Fv={bevel:.015,depth:.09,thickness:1,dispersion:.5,roughness:.05,tint:"#e7c574",tintDensity:3.2,highlight:"#e7c574"},Bv=2e3,kv=900;function zv(){let i=document.createElement("canvas");return i.style.cssText=`position:absolute;inset:0;width:100%;height:100%;display:block;pointer-events:none;touch-action:none;opacity:0;transition:opacity ${$d}ms ease`,i}function Hv(i,e,t){return Bd({canvas:e},{src:i.dataset.src,cellSize:st(i,"cellSize",13),cellAspect:st(i,"cellAspect",.9),contrast:st(i,"contrast",2.5),edgeContrast:st(i,"edgeContrast",3.3),environmentIntensity:st(i,"environmentIntensity",1.6),roughness:st(i,"roughness",.35),scale:st(i,"scale",5),floatIntensity:st(i,"floatIntensity",.3),rotationIntensity:st(i,"rotationIntensity",.12),floatSpeed:st(i,"floatSpeed",.7),fov:st(i,"fov",28),cameraDistance:st(i,"cameraDistance",9.2),autoRotateSpeed:st(i,"autoRotateSpeed",.5),ascii:!1,colored:!0,autoRotate:!0,orbit:!1,zoom:!1,highlight:kc(i,"highlight","#e7c574"),onLoad:t,onError:()=>i.setAttribute("data-mark3d-state","failed")})}function Vv(i,e,t){return qd({canvas:e},{src:i.dataset.glassSrc||i.dataset.src,scale:st(i,"glassScale",4.2),fov:st(i,"glassFov",50),cameraDistance:st(i,"glassCameraDistance",5),floatSpeed:st(i,"glassFloatSpeed",.1),floatIntensity:st(i,"glassFloatIntensity",.4),rotationIntensity:st(i,"glassRotationIntensity",.15),autoRotate:!0,autoRotateSpeed:st(i,"glassAutoRotateSpeed",.5),ior:st(i,"glassIor",1.6),clearcoat:st(i,"glassClearcoat",.5),environmentIntensity:st(i,"glassEnvironmentIntensity",1),tint:kc(i,"glassTint",Ti.tint),tintDensity:st(i,"glassTintDensity",Ti.tintDensity),highlight:kc(i,"glassHighlight",Ti.highlight),backgroundImage:kc(i,"glassBackgroundImage",""),orbit:i.dataset.orbit!=="false",zoom:!1,bevel:st(i,"glassBevel",Ti.bevel),depth:st(i,"glassDepth",Ti.depth),thickness:st(i,"glassThickness",Ti.thickness),dispersion:st(i,"glassDispersion",Ti.dispersion),roughness:st(i,"glassRoughness",Ti.roughness),onLoad:t,onError:()=>i.setAttribute("data-mark3d-state","failed")})}var ou=class{constructor(e,t,n,s,r){this.host=e;this.button=t;this.navMark=n;this.veil=s;this.scroller=r;Dt(this,"instance",null);Dt(this,"canvas");Dt(this,"mode");Dt(this,"progress",-1);Dt(this,"appliedStep",-1);Dt(this,"hovered",!1);Dt(this,"idleTimer",0);Dt(this,"returnRaf",0);Dt(this,"hideTimer",0);Dt(this,"frozen",!1);Dt(this,"easeToBase",null);Dt(this,"anchor",{left:0,top:0});Dt(this,"startScale",1);Dt(this,"startTop",90);Dt(this,"travel",600);Dt(this,"ticking",!1);Dt(this,"reduced");Dt(this,"onScroll",()=>{this.ticking||this.reduced.matches||(this.ticking=!0,requestAnimationFrame(()=>{this.ticking=!1,this.apply(this.scrollProgress())}))});Dt(this,"onResize",()=>{this.measure(),this.apply(this.reduced.matches?1:this.scrollProgress())});this.mode=e.dataset.mark3dMode||"glass",this.reduced=window.matchMedia("(prefers-reduced-motion: reduce)"),this.canvas=zv(),e.appendChild(this.canvas),this.measure(),this.apply(this.reduced.matches?1:this.scrollProgress());let o=this.mode==="glass"?Vv:Hv;this.instance=o(e,this.canvas,()=>{requestAnimationFrame(()=>requestAnimationFrame(()=>{this.canvas.style.opacity="1",e.setAttribute("data-mark3d-state","ready"),this.wireOrbitReturn()}))}),(this.scroller??window).addEventListener("scroll",this.onScroll,{passive:!0}),addEventListener("resize",this.onResize,{passive:!0}),this.button&&(this.button.addEventListener("click",()=>this.scrollToTop()),this.button.addEventListener("mouseenter",()=>this.setHover(!0)),this.button.addEventListener("mouseleave",()=>this.setHover(!1)),this.button.addEventListener("focus",()=>this.setHover(!0)),this.button.addEventListener("blur",()=>this.setHover(!1)))}scrollToTop(){let e=this.scroller,t=e?e.scrollTop:scrollY;if(t<=0)return;if(this.reduced.matches){e?e.scrollTop=0:scrollTo(0,0);return}let n=performance.now(),s=jd(t/4e3)*400+420,r=()=>{let o=Math.min(1,(performance.now()-n)/s),a=t*(1-Bc(o));e?e.scrollTop=a:scrollTo(0,a),o<1&&requestAnimationFrame(r)};requestAnimationFrame(r)}setHover(e){this.hovered=e,this.paintOpacity()}setFrozen(e){this.instance?.setOptions?.(e?{autoRotate:!1,floatIntensity:0,rotationIntensity:0}:{autoRotate:!0,floatIntensity:st(this.host,"glassFloatIntensity",.4),rotationIntensity:st(this.host,"glassRotationIntensity",.15)}),this.host.setAttribute("data-mark3d-frozen",String(e)),this.canvas.style.opacity=e?"0":"1",clearTimeout(this.hideTimer),e?this.hideTimer=setTimeout(()=>{this.canvas.style.display="none"},$d+60):this.canvas.style.display="",e&&this.easeToBase?.(!0)}wireOrbitReturn(){let e=this.instance,t=e?.camera,n=e?.controls;if(!t||!n)return;let s=()=>{let a=n.target,c=t.position.x-a.x,l=t.position.y-a.y,h=t.position.z-a.z,u=Math.hypot(c,l,h)||1e-6;return{r:u,phi:Math.acos(Math.min(1,Math.max(-1,l/u))),theta:Math.atan2(c,h)}},r=(a,c,l)=>{let h=n.target,u=a*Math.sin(c);t.position.set(h.x+u*Math.sin(l),h.y+a*Math.cos(c),h.z+u*Math.cos(l)),n.update()},o=s();this.easeToBase=a=>{cancelAnimationFrame(this.returnRaf);let c=s(),l=o.theta-c.theta;for(;l>Math.PI;)l-=2*Math.PI;for(;l<-Math.PI;)l+=2*Math.PI;let h=performance.now(),u=()=>{let f=Math.min(1,(performance.now()-h)/kv),d=Bc(f),p=s();r(p.r,Rs(c.phi,o.phi,d),a?c.theta+l*d:p.theta),f<1&&(this.returnRaf=requestAnimationFrame(u))};u()},n.addEventListener("start",()=>{clearTimeout(this.idleTimer),cancelAnimationFrame(this.returnRaf)}),n.addEventListener("end",()=>{clearTimeout(this.idleTimer),this.idleTimer=setTimeout(()=>this.easeToBase?.(this.frozen),Bv)})}fit(){let e=(document.querySelector(".landing-eyebrow")??document.querySelector(".landing-copy"))?.getBoundingClientRect(),t=st(this.host,"startTop",8),n=Math.max(96,(e?.top??innerHeight*.6)-t-12);this.startScale=Math.min(1,n/616),this.startTop=t}measure(){this.fit();let e=this.navMark?.getBoundingClientRect();e&&(this.anchor={left:st(this.host,"endInsetX",70),top:e.top});let t=document.querySelector(".landing-hero"),n=t?t.getBoundingClientRect().height:innerHeight;this.travel=Math.max(240,Math.min(620,n*.65)),this.host.style.left=`${this.anchor.left}px`,this.host.style.top=`${this.anchor.top}px`,this.button&&(this.button.style.left=`${this.anchor.left}px`,this.button.style.top=`${this.anchor.top}px`)}scrollProgress(){let e=this.scroller?this.scroller.scrollTop:scrollY;return jd(e/this.travel)}paintOpacity(){let e=Bc(Math.max(this.progress,0));this.host.style.opacity=String(this.hovered?.9:Rs(1,.5,e))}apply(e){if(Math.abs(e-this.progress)<.001)return;this.progress=e;let t=Bc(e),n=Rs(this.startScale,1/Uv,t),r=(innerWidth/2-this.host.offsetWidth*this.startScale/2+st(this.host,"startOffsetX",-400)-this.anchor.left)*(1-t),o=(this.startTop-this.anchor.top)*(1-t);this.host.style.transform=`translate(${r}px, ${o}px) scale(${n})`,this.paintOpacity();let a=f=>(f/n).toFixed(1),c=Rs(.06,.5,t).toFixed(3),l=Rs(.1,.36,t).toFixed(3);if(this.host.style.filter=`drop-shadow(0 0 ${a(.8)}px rgba(24,28,22,${c})) drop-shadow(0 ${a(Rs(3,4,t))}px ${a(Rs(9,7,t))}px rgba(24,28,22,${l}))`,this.mode==="glass"&&this.instance?.setOptions){let f=t>=.45?1:0;f!==this.appliedStep&&(this.appliedStep=f,this.instance.setOptions(f?Fv:Ti))}this.veil&&(this.veil.style.opacity=String(t));let h=t>.9;h!==this.frozen&&(this.frozen=h,this.setFrozen(h));let u=t<=.6;this.host.style.pointerEvents=u?"auto":"none",this.canvas.style.pointerEvents=u?"auto":"none",this.button&&(this.button.hidden=u)}};function Jd(){Kd();let i=document.querySelector("[data-hausv-mark-3d]");if(!i||!i.dataset.src)return;let e=Number(i.dataset.minWidth??0);e&&!matchMedia(`(min-width: ${e}px)`).matches||Ov()&&new ou(i,document.querySelector("[data-mark3d-top]"),document.querySelector(".landing-mark"),document.querySelector(".mark3d-veil"),document.querySelector(".bend-content"))}document.readyState==="loading"?document.addEventListener("DOMContentLoaded",Jd,{once:!0}):Jd();export{Jd as initHausvMark3d};
/*! Bundled license information:

three/build/three.core.js:
three/build/three.module.js:
  (**
   * @license
   * Copyright 2010-2026 Three.js Authors
   * SPDX-License-Identifier: MIT
   *)
*/
