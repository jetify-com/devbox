# Devbox 📦

### 即时、简单、可预测地创建Shell与Container

[![Join Discord](https://img.shields.io/discord/903306922852245526?color=7389D8&label=discord&logo=discord&logoColor=ffffff)](https://discord.gg/agbskCJXk2) ![License: Apache 2.0](https://img.shields.io/github/license/jetify-com/devbox) [![version](https://img.shields.io/github/v/release/jetify-com/devbox?color=green&label=version&sort=semver)](https://github.com/jetify-com/devbox/releases) [![tests](https://github.com/jetify-com/devbox/actions/workflows/tests.yaml/badge.svg)](https://github.com/jetify-com/devbox/actions/workflows/tests.yaml)

---

## 它是什么?

Devbox是一个可以让你轻松地创建隔离环境的shell与container的命令行工具。首先定义你开发环境中所需的软件包列表，随后Devbox使用该定义来为你的应用程序创建一个隔离的环境。

在实践中，Devbox的工作方式类似于像`yarn`这样的软件包管理器--只不过它所管理的软件包是操作系统级别的。（这些包你通常会通过`brew`和`apt-get`来进行安装）。

Devbox最初由[Jetify](https://www.jetify.com)进行开发，其内部由`nix`驱动。

## 示例
下面的例子创建了一个带有`python 2.7`和`go 1.18`的开发环境，尽管这些包并没有在底层机器中被安装。

![screen cast](https://user-images.githubusercontent.com/279789/186491771-6b910175-18ec-4c65-92b0-ed1a91bb15ed.svg)


## 好处

### 为团队中的每一个人提供一个统一的Shell

通过`devbox.json`文件来声明项目中所需要的工具列表，并运行`devbox shell`。这样，参与项目工作的每一个人都会获得一个与这些工具完全版本的shell环境。

### 尝试新工具而不污染原先配置的环境

由Devbox创建的开发环境与你的笔记本电脑中的其他东西是隔离的。有什么工具你想尝试，却又不想把环境弄得一团糟？可以把这个工具添加到Devbox的shell中，而当你不再需要它的时候，就可以把它删除--同时保持你的笔记本电脑始终是原始的状态。

### 不以牺牲速度为代价

Devbox可以在你的笔记本电脑上直接创建隔离环境，而不需要额外的虚拟化以至于使得你的文件系统或每个命令都变得缓慢。当你准备打包时，就可以把它变成一个等效的container。

### 同版本冲突说再见

你是否正在处理多个项目，而所有这些项目都需要同一个二进制文件的不同版本？与其尝试在你的笔记本电脑上安装同一二进制文件的冲突版本，不如为每个项目创建一个隔离环境，并为每个项目使用你想要的任何版本。

### 瞬间将你的应用程序变成一个容器

Devbox分析你的源代码并立即将其转化为可以部署到任何云中、并符合OCI标准的镜像。该镜像在速度、大小、安全和缓存方面都进行了优化......而且不需要编写`Dockerfile`。而且与[buildpacks](https://buildpacks.io/)不同的是，devbox处理起来更快。

### 不要再重复声明依赖关系

当你在笔记本电脑上开发时，以及当你把它打包成一个容器准备部署到云端时，你的应用程序往往需要相同的依赖关系集。Devbox的开发环境是同构的：这意味着我们可以把它们变成本地的Shell环境或云端的container，所有这些都不需要重复完成。

## 安装Devbox

除了安装Devbox本身之外，你还需要安装`nix`和`docker`，因为Devbox依赖于它们。

1. 安装 [Nix Package Manager](https://nixos.org/download.html)。(别担心，你不需要学习Nix。)

2. 安装[Docker Engine](https://docs.docker.com/engine/install/)或[Docker Desktop](https://www.docker.com/get-started/)。注意，只有当你想创建容器时才需要docker--如果没有它，shell功能也能工作。

3. 安装Devbox:

   ```sh
   curl -fsSL https://get.jetify.com/devbox | bash
   ```

## 快速入门：快速又确定的shell

在这个快速入门中，我们将创建一个安装了特定工具的开发shell。这些工具只有在使用这个Devbox shell时才能使用，以确保我们不会污染你的机器。

1. 在一个新的空文件夹中打开一个终端。

2. 初始化Devbox:

   ```bash
   devbox init
   ```

   这将在当前目录下创建一个`devbox.json`文件。你应该把它提交到源码控制里。

3. 从[Nix Packages](https://search.nixos.org/packages)添加命令行工具。例如，要添加Python 3.10：

   ```bash
   devbox add python310
   ```
4. 你的`devbox.json`文件记录了你所添加的软件包，它现在应该看起来是这样的：

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

    你可以看出你是在Devbox shell中（而不是你的普通终端），因为shell的提示和目录已经改变。

6. 使用你喜欢的工具。

   In this example we installed Python 3.10, so let’s use it.

   ```bash
   python --version
   ```

7. 你的常规工具也是可用的，包括环境变量和配置设置。

   ```bash
   git config --get user.name
   ```

8. 要退出Devbox shell并返回到你的常规shell：

   ```bash
   exit
   ```

## 快速入门：迅速的Docker镜像

Devbox使得将你的应用程序打包成一个符合OCI标准的容器镜像变得很容易。Devbox会分析你的代码，自动识别你的项目所需的正确工具链，并将其构建为一个docker镜像。

1. 使用`devbox init`来初始化你的项目，如果还未初始化的话。

2. 构建镜像:

   ```bash
   devbox build
   ```

   生成的镜像名叫 `devbox`.

3. 用一个更具体的名称来标记该镜像：

   ```bash
   docker tag devbox my-image:v0.1
   ```
### 自动检测的语言
Devbox目前支持检测以下两种语言：

- Go
- Python (Poetry)

想要支持更多的语言？[Ask for a new Language](https://github.com/jetify-com/devbox/issues) 或通过Pull Request贡献一个。

## 额外命令

`devbox help` - 用来查看所有的命令

`devbox plan` - 用来查看Devbox在生成container时的配置与步骤

## 加入我们的开发者社区

+ 通过加入[Jetify Discord Server](https://discord.gg/agbskCJXk2)来与我们聊天 - 我们有一个#devbox频道专门用于这个项目。
+ 使用[Github Issues](https://github.com/jetify-com/devbox/issues)提交错误报告和功能请求。
+ 在[Jetify’s Twitter](https://twitter.com/jetify_com)上关注我们的产品更新。

## 相关工作

感谢[Nix](https://nixos.org/)所提供的独立的shell。

## License

本项目在[Apache 2.0 License](https://github.com/jetify-com/devbox/blob/main/LICENSE)下自豪地开放源代码。
