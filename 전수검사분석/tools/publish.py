# 전수검사분석 새 버전 배포용 스크립트 (관리자용)
#
#   화면만 바꾼 경우:
#     python 전수검사분석/tools/publish.py 1.5.0 "바뀐 내용 1" "바뀐 내용 2"
#   실행 파일(exe)도 바꾼 경우 (release/HBS_Inspection.exe 를 먼저 새 파일로 교체):
#     python 전수검사분석/tools/publish.py 1.5.0 "바뀐 내용" --launcher 1.5.0
#
# 하는 일
#  - 전수검사분석/app.html 의 버전 표시를 바꾼다
#  - 파일 해시를 계산해 전수검사분석/version.json 을 새로 쓴다  (v1.4.0 이후 프로그램이 읽는 곳)
#  - 저장소 맨 위 version.json 도 같이 쓴다  (v1.3.0 이하 프로그램이 읽는 곳 → 새 폴더로 안내)
# 그 다음 git commit / push 하면 사용자 프로그램이 켜질 때 자동으로 새 버전을 받는다.
import hashlib, json, re, sys, datetime, os

FOLDER = "전수검사분석"
prog = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))   # .../전수검사분석
repo = os.path.dirname(prog)
args = sys.argv[1:]
if not args or not re.fullmatch(r"\d+(\.\d+)*", args[0]):
    sys.exit('사용법: python 전수검사분석/tools/publish.py 1.5.0 "변경 내용" [--launcher 1.5.0]')
ver = args[0]
launcher = None
if "--launcher" in args:
    i = args.index("--launcher"); launcher = args[i + 1]; del args[i:i + 2]
notes = args[1:]

ap = os.path.join(prog, "app.html")
html = open(ap, encoding="utf-8").read()
html, n = re.subn(r'<meta name="app-version" content="[0-9.]+">', f'<meta name="app-version" content="{ver}">', html)
if n != 1:
    sys.exit("app.html 에서 버전 표시를 찾지 못했습니다")
open(ap, "w", encoding="utf-8", newline="\n").write(html)

vp = os.path.join(prog, "version.json")
old = json.load(open(vp, encoding="utf-8")) if os.path.exists(vp) else {}
sha = lambda p: hashlib.sha256(open(p, "rb").read()).hexdigest()
v = {
    "program": FOLDER,
    "version": ver,
    "date": datetime.date.today().isoformat(),
    "notes": notes or old.get("notes", []),
    "app": "app.html",
    "app_sha256": sha(ap),
    "launcher": launcher or old.get("launcher", "1.4.0"),
    "exe": "release/HBS_Inspection.exe",
    "exe_sha256": sha(os.path.join(prog, "release", "HBS_Inspection.exe")),
}
json.dump(v, open(vp, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
legacy = dict(v, app=f"{FOLDER}/app.html", exe=f"{FOLDER}/release/HBS_Inspection.exe",
              note="v1.3.0 이하 전수검사분석 프로그램용 안내 파일입니다. 실제 배포 파일은 전수검사분석 폴더에 있습니다.")
json.dump(legacy, open(os.path.join(repo, "version.json"), "w", encoding="utf-8"), ensure_ascii=False, indent=2)
print(json.dumps(v, ensure_ascii=False, indent=2))
