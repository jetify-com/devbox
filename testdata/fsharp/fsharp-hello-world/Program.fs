// For more information see https://aka.ms/fsharp-console-apps

[<EntryPoint>]
let main args =
    let dotNetVersion = System.Runtime.InteropServices.RuntimeInformation.FrameworkDescription
    printfn "Installed version is %A" dotNetVersion

    let expectedVersionPrefix = ".NET 6"
    if (not (dotNetVersion.StartsWith(expectedVersionPrefix))) then
      raise (System.Exception(sprintf "Expected version %A but got version %A" expectedVersionPrefix dotNetVersion))
    else 
      0
