To make a csharp testdata, a few things to know:
1. C# language runs on the DotNet Framework.
2. Specific Versions of DotNet support a C# version AND older C# versions.
  - For example: dotnet6.0 supports C# 10, but also C# 9 and likely even older versions.
  - reference: https://docs.microsoft.com/en-us/dotnet/csharp/language-reference/configure-language-version#defaults

The way I created a folder was by doing the following:
1. Create a dummy folder: `dummy/` and `devbox init` inside it. Do `devbox add dotnet-sdk_5`.
  - Replace `dotnet-sdk_5` with the version of dotnet you want. Get the exact nixpkg name from `search.nixos.org`.
2. Then do `devbox shell` to get a shell with that `dotnet`.
3. Then do: `dotnet new console -o csharp_9-dotnet_5`
  - Replace `csharp_9-dotnet_5` with whatever (dotnet, C# language version) you are selecting.
  - `dotnet run` should print "Hello World!"
  - To create a project with a specific language version, do:
    - `dotnet new console -o <name> --langVersion 9.1` to use C# 9.1. The default langVersion is that dotNet version's default.
4. Then do: `dotnet new gitignore` and `git add <folder>/.gitignore`
  - This eliminates the `obj` folder that `dotnet new console` will generate.
