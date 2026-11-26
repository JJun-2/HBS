# 전수검사분석 — 전수검사 데이터 분석 프로그램

(주)한부솔루션 제품 전수검사 데이터를 누적해 대시보드와 보고서로 보는 윈도우 프로그램입니다. Made By J

## 내려받기 (처음 한 번)

[HBS_Inspection_setup.zip 내려받기](https://github.com/JJun-2/HBS/raw/main/전수검사분석/release/HBS_Inspection_setup.zip)

압축을 풀고 `전수검사분석.exe` 를 실행합니다. 설치 과정은 없고, 인터넷 없이도 동작합니다.
그다음부터는 프로그램을 켤 때 이 폴더를 확인해서 새 버전을 자동으로 받아 적용합니다.

## 자동 업데이트 동작

1. 프로그램이 켜지면 창을 띄우기 전에 `전수검사분석/version.json` 을 읽습니다 (인터넷이 안 되면 몇 초 안에 건너뜀). 켜 둔 동안에는 다시 확인하지 않습니다.
2. `version` 이 지금보다 높으면 `app.html`(화면)을 받아 SHA-256 을 확인한 뒤 `data\app\app.html` 에 저장하고, 새 화면으로 바로 엽니다.
3. `launcher` 가 지금 실행 파일 버전보다 높으면 `release/HBS_Inspection.exe` 를 받아 확인한 뒤 실행 파일을 바꿉니다. 다음 실행부터 적용됩니다.
4. `settings.json`(있을 때)은 관리자가 정한 배포용 기본 설정입니다. 각 사용자가 직접 바꾼 설정이 우선합니다.
5. 검사 데이터(`data\inspection_data.json`)는 업데이트와 상관없이 그대로 유지됩니다.

## 관리자: 새 버전 배포하는 법

화면만 고친 경우 (대부분):

```
# 전수검사분석/app.html 수정 후 (저장소 맨 위에서)
python 전수검사분석/tools/publish.py 1.5.0 "바뀐 내용 1" "바뀐 내용 2"
git add -A && git commit -m "전수검사분석 v1.5.0" && git push
```

실행 파일도 바꾼 경우: `src/` 에서 다시 빌드해 `release/HBS_Inspection.exe` 를 교체하고 `--launcher 1.5.0` 을 붙입니다.
처음 배포용 `release/HBS_Inspection_setup.zip` 도 같이 바꿔 두세요.

```
cd 전수검사분석/src
cp ../app.html .
python mkicon.py mark.png   # 회사 로고 H 마크로 아이콘 리소스(rsrc_windows_amd64.syso) 생성
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o ../release/HBS_Inspection.exe .
```

배포용 기본 설정: 프로그램 [기본 설정] → [관리자] → [배포용 설정] → `settings.json 으로 내보내기` 한 파일을 이 폴더에 올립니다.

## 파일 구성

| 경로 (전수검사분석/ 아래) | 내용 |
|---|---|
| `version.json` | 최신 버전 정보와 파일 해시 (프로그램이 읽음) |
| `app.html` | 프로그램 화면 (자동 업데이트 대상) |
| `settings.json` | 배포용 기본 설정 (선택) |
| `release/HBS_Inspection.exe` | 실행 파일 (자동 업데이트 대상) |
| `release/HBS_Inspection_setup.zip` | 처음 설치용 묶음 |
| `src/` | 실행 파일 소스 (Go) |
| `tools/publish.py` | 버전 올리기 스크립트 |

저장소 맨 위의 `version.json` 은 v1.3.0 이하 프로그램이 새 폴더로 넘어오도록 남겨 둔 안내용 파일입니다. publish.py 가 같이 갱신합니다.
