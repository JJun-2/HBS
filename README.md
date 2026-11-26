# HBS 전수검사 데이터 분석 프로그램

(주)한부솔루션 제품 전수검사 데이터를 누적해 대시보드와 보고서로 보는 윈도우 프로그램입니다. Made By J

## 내려받기 (처음 한 번)

[HBS_Inspection_setup.zip 내려받기](https://github.com/JJun-2/HBS/raw/main/release/HBS_Inspection_setup.zip)

압축을 풀고 `전수검사분석.exe` 를 실행합니다. 설치 과정은 없고, 인터넷 없이도 동작합니다.
그다음부터는 프로그램이 켜질 때 이 저장소를 확인해서 새 버전을 자동으로 받습니다.

## 자동 업데이트 동작

1. 프로그램이 켜지면 `version.json` 을 읽습니다 (인터넷이 안 되면 건너뜀).
2. `version` 이 지금보다 높으면 `app.html`(화면)을 받아 SHA-256 을 확인한 뒤 `data\app\app.html` 에 저장하고, 화면 위에 [지금 적용] 안내를 띄웁니다.
3. `launcher` 가 지금 실행 파일 버전보다 높으면 `release/HBS_Inspection.exe` 를 받아 확인한 뒤 실행 파일을 바꿉니다. 다음 실행부터 적용됩니다.
4. 검사 데이터(`data\inspection_data.json`)는 업데이트와 상관없이 그대로 유지됩니다.

## 관리자: 새 버전 배포하는 법

화면만 고친 경우 (대부분):

```
# app.html 수정 후
python tools/publish.py 1.2.0 "바뀐 내용 1" "바뀐 내용 2"
git add -A && git commit -m "v1.2.0" && git push
```

실행 파일도 바꾼 경우: `src/` 에서 다시 빌드해 `release/HBS_Inspection.exe` 를 교체하고 `--launcher 1.2.0` 을 붙입니다.
처음 배포용 `release/HBS_Inspection_setup.zip` 도 같이 바꿔 두세요.

```
cd src
cp ../app.html .
python mkicon.py mark.png   # 회사 로고 H 마크로 아이콘 리소스(rsrc_windows_amd64.syso) 생성
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o ../release/HBS_Inspection.exe .
```

## 파일 구성

| 경로 | 내용 |
|---|---|
| `version.json` | 최신 버전 정보와 파일 해시 (프로그램이 읽음) |
| `app.html` | 프로그램 화면 (자동 업데이트 대상) |
| `release/HBS_Inspection.exe` | 실행 파일 (자동 업데이트 대상) |
| `release/HBS_Inspection_setup.zip` | 처음 설치용 묶음 |
| `src/` | 실행 파일 소스 (Go) |
| `tools/publish.py` | 버전 올리기 스크립트 |
