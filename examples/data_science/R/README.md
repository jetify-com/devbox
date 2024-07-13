# R

[R is a free software environment for statistical computing and graphics](https://www.r-project.org/).

[**Example Repo**](https://github.com/jetify-com/devbox/tree/main/examples/data_science/R)

## Adding R to your Project

`devbox add R@4.4.1`, or in your `devbox.json` add:

```json
  "packages": [
    "R@4.4.1"
  ],
```

This will install R in your shell. You can find other versions of R by running `devbox search R`.
You can also view the available versions on [Nixhub](https://www.nixhub.io/packages/R).

## Installing Packages

[CRAN](https://cran.r-project.org/) is the main repository of R packages.
All of the CRAN packages are also available on [Nixhub](https://www.nixhub.io/).

You can install packages by running `devbox add rPackages.package_name`, where `package_name` is the name of the package you would normally install with `install.packages()`.
Note that for packages with a dot in the name you will need to replace the dot with an underscore, i.e. `data.table` -> `data_table` (see example below).

```json
{
    "packages": [
      "R@4.4.1",
      "rPackages.data_table@latest",
      "rPackages.ggplot2@latest",
      "rPackages.tidyverse@latest"
    ],
}
```

You can access these packages in your R scripts as usual with `library(data.table)``.

## Example script

In this [example repo](https://github.com/jetify-com/devbox/tree/main/examples/data_science/R), after running `devbox shell`, you can start an R repl with `R` then create an example plot with `source("src/examplePlot.R")`. 
Alternatively run `Rscript src/examplePlot.R`.
This will create an `Rplots.pdf` file.

## Troubleshooting

If you get warnings like:

> During startup - Warning messages:
> 1: setting LC_CTYPE failed, using "C"
> 2: Setting LC_COLLATE failed, using "C"
> ...

then you need to set your locale.
Find your locale (outside of a devbox shell) using `locale` in your terminal. You will see something like:

> LANG=en_NZ.UTF-8
> LANGUAGE=en_NZ:en
> LC_CTYPE="en_NZ.UTF-8"
> ...

To set your locale, edit the `init_hook` array in the shell object in `devbox.json` to export two environment variables like below (using your specific locale):

```json
{
  "shell": {
    "init_hook": [
      "export LANG=en_NZ.UTF-8",
      "export LC_ALL=en_NZ.UTF-8"
    ]
  }
}
```
