import torch

def create_arrays(n):
    x = torch.ones(n, n)
    y = torch.randn(n, n * 2)
    return x , y


def main():
    x, y = create_arrays(1000)
    x = x.to("cuda")
    y = y.to("cuda")
    z = x @ y
    print(z)

if __name__ == "__main__":
    main()