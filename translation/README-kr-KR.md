# Devbox 📦

### 즉각적이고, 쉽고, 예측 가능한 개발 환경

[![Join Discord](https://img.shields.io/discord/903306922852245526?color=7389D8&label=discord&logo=discord&logoColor=ffffff)](https://discord.gg/jetify) ![License: Apache 2.0](https://img.shields.io/github/license/jetify-com/devbox) [![version](https://img.shields.io/github/v/release/jetify-com/devbox?color=green&label=version&sort=semver)](https://github.com/jetify-com/devbox/releases) [![tests](https://github.com/jetify-com/devbox/actions/workflows/cli-post-release.yml/badge.svg)](https://github.com/jetify-com/devbox/actions/workflows/cli-release.yml?branch=main) [![Built with Devbox](https://www.jetify.com/img/devbox/shield_galaxy.svg)](https://www.jetify.com/devbox/docs/contributor-quickstart/)

## 무엇인가요?

[Devbox](https://www.jetify.com/devbox/)는 개발을 위한 격리된 셸을 쉽게 만들 수 있는 명령줄 도구 (command-line tool) 입니다. 개발 환경에 필요한 패키지 목록을 정의하는 것으로 시작하면 Devbox가 해당 정의를 사용하여 애플리케이션 전용 격리 환경을 생성합니다. 

실제로 Devbox는 `yarn`과 같은 패키지 관리자와 유사하게 작동하지만, 관리하는 패키지가 운영 체제 수준(일반적으로 `brew` 또는 `apt-get`으로 설치하는 것과 같은 종류)에 있다는 점이 다릅니다. Devbox를 사용하면 Nix 패키지 레지스트리에서 [400,000개 이상의 패키지 버전](https://www.nixhub.io)을 설치할 수 있습니다. 

Devbox는 원래 [Jetify](https://www.jetify.com)에서 개발되었으며 내부적으로 `nix`로 구동됩니다.  

## 데모

아래 예제는 기본 머신에 해당 패키지가 설치되어 있지 않더라도 `python 2.7`과 `go 1.18`로 개발 환경을 생성합니다:

![screen cast](https://user-images.githubusercontent.com/279789/186491771-6b910175-18ec-4c65-92b0-ed1a91bb15ed.svg)

## Devbox 설치하기

다음 설치 스크립트를 사용하여 최신 버전의 Devbox를 설치하세요:

```sh
curl -fsSL https://get.jetify.com/devbox | bash
```

자세한 내용은 [Devbox 문서](https://www.jetify.com/devbox/docs/installing_devbox/)를 참조하세요.

## 혜택

### 팀원 모두를 위한 일관된 셸

프로젝트에 필요한 도구 목록을 `devbox.json` 파일을 통해 선언하고 `devbox shell`을 실행하세요. 프로젝트에 참여하는 모든 사람이 정확히 동일한 버전의 도구가 포함된 셸 환경을 갖게 됩니다.

### 노트북을 어지럽히지 않고 새로운 도구를 사용해 보세요

Devbox로 생성된 개발 환경은 노트북의 다른 모든 것과 격리되어 있습니다. 노트북을 엉망으로 만들지 않고 사용해 보고 싶은 도구가 있나요? 그 도구를 Devbox 셸에 추가하고, 더 이상 필요하지 않을 시 제거하면 노트북을 깔끔하게 유지할 수 있습니다.

### 속도를 희생하지 마세요

Devbox는 파일 시스템이나 모든 명령의 속도를 저하시키는 추가 가상화 계층 없이도 노트북에서 바로 격리된 환경을 만들 수 있습니다. 출시 준비가 완료되면 동등한 컨테이너로 전환되지만 그 전에는 그렇지 않습니다.

### 버젼 충돌 문제는 이제 안녕

동일한 바이너리의 다른 버전이 필요한 여러 프로젝트에서 작업하고 계신가요? 노트북에 동일한 바이너리의 충돌하는 버전을 설치하는 대신 각 프로젝트에 대해 격리된 환경을 만들고 각각에 원하는 버전을 사용하세요.

### 개발 환경을 휴대하세요

Devbox의 개발 환경은 *이동성*이 있습니다. 환경을 정확히 한 번만 선언하고 그 단일 정의를 다음과 같은 여러 가지 방법으로 사용할 수 있습니다:

+ `devbox shell`을 통해 생성된 로컬 셸
+ VSCode와 함께 사용할 수 있는 개발 컨테이너 (devcontainer)
+ 개발에 사용한 것과 동일한 도구로 프로덕션 이미지를 빌드할 수 있는 도커 파일(Dockerfile) 프로덕션 이미지를 빌드할 수 있습니다.
+ 로컬 환경을 미러링하는 클라우드의 원격 개발 환경.

## Quickstart: 빠르고 결정론적인 셸 만들어보기

이 퀵스타트 가이드 에서는 특정 도구가 설치된 개발 셸을 만들어 보겠습니다. 이러한 도구는 이 Devbox 셸을 사용할 때만 사용할 수 있으므로 컴퓨터를 어지럽히지 않습니다.

1. 새 빈 폴더에서 터미널을 엽니다.

2. Devbox를 초기화합니다:

   ```bash
   devbox init
   ```

   이렇게 하면 현재 디렉터리에 `devbox.json` 파일이 생성됩니다. 이 파일을 소스 제어에 커밋해야 합니다.

3. Nix에서 명령줄 도구를 추가합니다. 예를 들어 Python 3.10을 추가하려면:

   ```bash
   devbox add python@3.10
   ```

   [Nixhub.io](https://www.nixhub.io)에서 더 많은 패키지를 검색하세요.

4. 이제 `devbox.json` 파일은 추가한 패키지를 추적하며, 다음과 같이 보일 것입니다:

   ```json
   {
      "packages": [
         "python@3.10"
       ]
   }
   ```

5. 이러한 도구가 설치된 새 셸을 시작합니다:

   ```bash
   devbox shell
   ```

   셸 프롬프트가 변경되었으므로 일반 터미널이 아닌 Devbox 셸에 있다는 것을 알 수 있습니다.

6. 선호하는 도구를 사용합니다.

   이 예제에서는 Python 3.10을 설치했으므로 이를 사용해 보겠습니다.

   ```bash
   python --version
   ```

7. 환경 변수 및 구성 설정을 포함한 일반 도구도 사용할 수 있습니다.

   ```bash
   git config --get user.name
   ```

8. Devbox 셸을 종료하고 일반 셸로 돌아가려면:

   ```bash
   exit
   ```

자세한 내용은 [Devbox 문서 퀵스타트](https://www.jetify.com/devbox/docs/quickstart/)를 참조하세요.

## 추가 명령어들

`devbox help` - 모든 명령어 보기

전체 명령어 목록은 [CLI Reference](https://www.jetify.com/devbox/docs/cli_reference/devbox/)를 참조하세요.

## 개발자 커뮤니티에 가입하세요!

+ [Jetify 디스코드 서버](https://discord.gg/jetify)에 가입하여 이야기를 나누어보세요. – 이 프로젝트 전용 #devbox 채널이 있습니다.
+ [Github Issues](https://github.com/jetify-com/devbox/issues)를 사용하여 버그 리포트 및 기능 요청을 제출하세요.
+ [Jetify's Twitter](https://twitter.com/jetify_com)를 팔로우하여 제품 업데이트를 확인하세요. 

## 기여하기

Devbox는 오픈소스 프로젝트이므로 언제든지 기여를 환영합니다. 풀 리퀘스트를 제출하기 전에 [기여 가이드](../../CONTRIBUTING.md)를 읽어주세요. 

[Devbox 개발을 위한 README.md](../../devbox.md)

## 관련된 작업들

격리된 셸을 제공해 주신 [Nix](https://nixos.org/)에게 감사드립니다.

## 번역판

+ [English](https://github.com/jetify-com/devbox/blob/main/README.md)
+ [Chinese](README-zh-CN.md)

## 라이선스

이 프로젝트는 [Apache 2.0 License](https://github.com/jetify-com/devbox/blob/main/LICENSE) 하 의 자랑스러운 오픈소스입니다.

# Devbox 문서 번역 가이드

Devbox 문서 번역에 참여해 주셔서 감사합니다! 이 가이드는 Devbox 문서 번역에 기여하는 방법을 안내합니다.
