**Background Context**

To make a C# testdata example, some background context:
1. The C# language runs on the .Net Framework.
2. A specific version of .Net will support the then newest C# version and older C# versions.
    - For example: .Net 6.0 supports C# 10, but also C# 9 and older versions.
    - reference: https://docs.microsoft.com/en-us/dotnet/csharp/language-reference/configure-language-version#defaults

**Steps to create a new C# test case**

The way I created a csharp testcase folder was by doing the following:
1. Create a dummy folder: `dummy/` and call `devbox init` inside it. Then add the nix-pkg: `devbox add dotnet-sdk_5`.
    - Replace `dotnet-sdk_5` with the version of .Net you want. Get the exact nix-pkg name from `search.nixos.org`.
2. Then do `devbox shell` to get a shell with that `dotnet` nix pkg.
3. Then do: `dotnet new console -o csharp_9-dotnet_5 --use-program-main=true`
    - Replace `csharp_9-dotnet_5` with the specific (dotnet, C# language version) you are testing for.
    - `dotnet run` should print "Hello World!"
    - To create a project with a specific language version, do:
      - `dotnet new console -o <name> --langVersion 9.1` to use C# 9.1. The default langVersion is that dotNet version's default.
4. Add `.gitignore` with these contents:
```
bin/
obj/
```
Alternatively, do `dotnet new gitignore` to get the fully specified `.gitignore`.
5. Add a `devbox.json` file in the new folder by doing `devbox init` inside it.
