namespace csharp_10_dotnet_6;

public class VersionMismatchException : Exception
{
    public VersionMismatchException(string msg): base(msg)
    {
    }
}

class Program
{
    static void Main(string[] args)
    {
        string version = System.Runtime.InteropServices.RuntimeInformation.FrameworkDescription;
        Console.WriteLine("Installed version is: {0}", version);

        string expectedVersion = ".NET 6";
        if (!version.StartsWith(expectedVersion))
        {
            string msg = string.Format("Expected version '{0}', but got version: '{1}'", expectedVersion, version);
            throw new VersionMismatchException(msg);
        }
    }
}
