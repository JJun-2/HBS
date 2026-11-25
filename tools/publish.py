# 새 버전 배포용 스크립트 (관리자용)
#
#   화면만 바꾼 경우:
#     python tools/publish.py 1.2.0 "바뀐 내용 1" "바뀐 내용 2"
#   실행 파일(exe)도 바꾼 경우 (release/HBS_Inspection.exe 를 먼저 새 파일로 교체):
#     python tools/publish.py 1.2.0 "바뀐 내용" --launcher 1.2.0
#
# 하는 일: app.html 의 버전 표시를 바꾸고, 파일 해시를 계산해 version.json 을 새로 쓴다.
# 그 다음 git commit / push 하면 사용자 프로그램이 켜질 때 자동으로 새 버전을 받는다.
import hashlib, json, re, sys, datetime, os

root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
args = sys.argv[1:]
if not args or not re.fullmatch(r"\d+(\.\d+)*", args[0]):
    sys.exit('사용법: python tools/publish.py 1.2.0 "변경 내용" [--launcher 1.2.0]')
ver = args[0]
launcher = None
if "--launcher" in args:
    i = args.index("--launcher"); launcher = args[i + 1]; del args[i:i + 2]
notes = args[1:]

ap = os.path.join(root, "app.html")
html = open(ap, encoding="utf-8").read()
html, n = re.subn(r'<meta name="app-version" content="[0-9.]+">', f'<meta name="app-version" content="{ver}">', html)
if n != 1:
    sys.exit("app.html 에서 버전 표시를 찾지 못했습니다")
open(ap, "w", encoding="utf-8", newline="\n").write(html)

vp = os.path.join(root, "version.json")
old = json.load(open(vp, encoding="utf-8")) if os.path.exists(vp) else {}
sha = lambda p: hashlib.sha256(open(p, "rb").read()).hexdigest()
v = {
    "version": ver,
    "date": datetime.date.today().isoformat(),
    "notes": notes or old.get("notes", []),
    "app": "app.html",
    "app_sha256": sha(ap),
    "launcher": launcher or old.get("launcher", "1.1.0"),
    "exe": "release/HBS_Inspection.exe",
    "exe_sha256": sha(os.path.join(root, "release", "HBS_Inspection.exe")),
}
json.dump(v, open(vp, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
print(json.dumps(v, ensure_ascii=False, indent=2))
