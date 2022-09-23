namespace csharp_10_dotnet_6_with_package;

using Newtonsoft.Json;

class Product 
{
  public string Name;
  public DateTime Expiry;
  public string[] Sizes;

  public Product() 
  {
    Name = "";
    Sizes = new string[] {};
  } 
}

class Program
{
    static void Main(string[] args)
    {
        Product product = new Product();
        product.Name = "Apple";
        product.Expiry = new DateTime(2008, 12, 28);
        product.Sizes = new string[] { "Small" };

        string json = JsonConvert.SerializeObject(product);
        Console.WriteLine(string.Format("serialized json for {0} is {1}", product, json));
    }
}
