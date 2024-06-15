# Devbox 📦

### 即时、简单、可预测地创建 Shell 与 Container

[![Join Discord](https://img.shields.io/discord/903306922852245526?color=7389D8&label=discord&logo=discord&logoColor=ffffff)](https://discord.gg/jetify) ![License: Apache 2.0](https://img.shields.io/github/license/jetify-com/devbox) [![version](https://img.shields.io/github/v/release/jetify-com/devbox?color=green&label=version&sort=semver)](https://github.com/jetify-com/devbox/releases) [![tests](https://github.com/jetify-com/devbox/actions/workflows/cli-post-release.yml/badge.svg)](https://github.com/jetify-com/devbox/actions/workflows/cli-release.yml?branch=main) [![Built with Devbox](https://www.jetify.com/img/devbox/shield_galaxy.svg)](https://www.jetify.com/devbox/docs/contributor-quickstart/)

---

## 它是什么?

[Devbox](https://www.jetify.com/devbox/) 是一个可以让你轻松地创建隔离环境的 shell 与 container 的命令行工具。首先定义你开发环境中所需的软件包列表，随后 Devbox 使用该定义来为你的应用程序创建一个隔离的环境。

在实践中，Devbox 的工作方式类似于像 `yarn` 这样的软件包管理器——只不过它所管理的软件包是操作系统级别的。（这些包你通常会通过 `brew` 和 `apt-get` 来进行安装）。使用 Devbox，你可以从 Nix 软件包注册表中安装超过 [400,000 个软件包版本]((https://www.nixhub.io))。

Devbox最初由 [Jetify](https://www.jetify.com) 进行开发，其内部由 `nix` 驱动。

## 示例

你可以点击下面的按钮在浏览器中试用 Devbox：

[![Open In Devbox.sh](https://www.jetify.com/img/devbox/open-in-devbox.svg)](https://devbox.sh/new)

下面的例子创建了一个带有 `python 2.7` 和 `go 1.18` 的开发环境，尽管这些包并没有在底层机器中被安装。

![screen cast](https://user-images.githubusercontent.com/279789/186491771-6b910175-18ec-4c65-92b0-ed1a91bb15ed.svg)

## 安装 Devbox

使用以下安装脚本获取最新版本的 Devbox：

```sh
curl -fsSL https://get.jetify.com/devbox | bash
```

在 [Devbox 文档](https://www.jetify.com/devbox/docs/installing_devbox/)中阅读更多内容。

## 好处

### 为团队中的每一个人提供一个统一的 Shell

通过 `devbox.json` 文件来声明项目中所需要的工具列表，并运行 `devbox shell`。这样，参与项目工作的每一个人都会获得一个与这些工具完全版本的 shell 环境。

### 尝试新工具而不污染原先配置的环境

由 Devbox 创建的开发环境与你的笔记本电脑中的其他东西是隔离的。有什么工具你想尝试，却又不想把环境弄得一团糟？可以把这个工具添加到 Devbox 的 shell 中，而当你不再需要它的时候，就可以把它删除——同时保持你的笔记本电脑始终是原始的状态。

### 不以牺牲速度为代价

Devbox可以在你的笔记本电脑上直接创建隔离环境，而不需要额外的虚拟化以至于使得你的文件系统或每个命令都变得缓慢。当你准备打包时，就可以把它变成等效的 container。

### 同版本冲突说再见

你是否正在处理多个项目，而所有这些项目都需要同一个二进制文件的不同版本？与其尝试在你的笔记本电脑上安装同一个二进制文件的冲突版本，不如为每个项目创建一个隔离环境，并为每个项目使用你想要的任何版本。

### 随身携带你的开发环境

Devbox 的开发环境是*可移植的*。我们使您能够只声明一次环境，并以多种不同方式使用这个单一定义，包括：

+ 通过 `devbox shell` 创建的本地 shell
+ 可在 VSCode 中使用的开发容器
+ 一个 Dockerfile，这样你可以用与你开发时使用的完全相同的工具构建生产镜像
+ 在云端的远程开发环境，该环境与本地环境完全一致

## 快速入门：快速又确定的 shell

在这个快速入门中，我们将创建一个安装了特定工具的开发 shell。这些工具只有在使用这个 Devbox shell 时才能使用，以确保我们不会污染你的机器。

1. 在一个新的空文件夹中打开一个终端。

2. 初始化 Devbox:

   ```bash
   devbox init
   ```

   这将在当前目录下创建一个 `devbox.json` 文件。你应该把它提交到源码控制里。

3. 从 [Nix](https://search.nixos.org/packages) 添加命令行工具。例如，要添加Python 3.10：

   ```bash
   devbox add python310
   ```

   在 [Nixhub.io](https://www.nixhub.io) 上搜索更多软件包。
   
4. 你的 `devbox.json` 文件记录了你所添加的软件包，它现在应该看起来是这样的：

   ```json
   {
      "packages": [
         "python310"
       ]
   }
   ```

5. 启动一个安装了这些工具的新shell：

   ```bash
   devbox shell
   ```

    你可以看出你是在 Devbox shell 中（而不是你的普通终端），因为 shell 的提示和目录已经改变。

6. 使用你喜欢的工具。

   在这个例子中，我们安装了 Python 3.10，所以让我们使用它吧。

   ```bash
   python --version
   ```

7. 你的常规工具也是可用的，包括环境变量和配置设置。

   ```bash
   git config --get user.name
   ```

8. 要退出 Devbox shell 并返回到你的常规 shell：

   ```bash
   exit
   ```

在 [Devbox 文档快速入门](https://www.jetify.com/devbox/docs/quickstart/)中阅读更多内容。

## 额外命令

`devbox help`，用来查看所有的命令。

请参阅 [CLI 参考](https://www.jetify.com/devbox/docs/cli_reference/devbox/)以获取完整的命令列表。

## 加入我们的开发者社区

+ 通过加入 [Jetify Discord Server](https://discord.gg/jetify) 来与我们聊天——我们有一个 #devbox 频道专门用于这个项目。
+ 使用 [Github Issues](https://github.com/jetify-com/devbox/issues) 提交错误报告和功能请求。
+ 在 [Jetify’s Twitter](https://twitter.com/jetify_com) 上关注我们的产品更新。

## 贡献

Devbox 是一个开源项目，所以欢迎贡献。在提交拉取请求之前，请阅读[我们的贡献指南](../../CONTRIBUTING.md)。

[Devbox 开发 README](../../devbox.md)

## 相关工作

感谢 [Nix](https://nixos.org/) 所提供的独立的shell。

## 翻译

+ [韩文](README-ko-KR.md)

## 许可证

本项目在 [Apache 2.0 License](https://github.com/jetify-com/devbox/blob/main/LICENSE) 下自豪地开放源代码。
