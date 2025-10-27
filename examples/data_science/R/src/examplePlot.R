# Load the required libraries
library(data.table)
library(ggplot2)

# Create example data
data <- data.table(
  x = rnorm(100),  # 100 random numbers from a normal distribution
  y = rnorm(100)   # 100 random numbers from a normal distribution
)

# Plot the data using ggplot2
demoPlot <- ggplot(data, aes(x = x, y = y)) +
  geom_point() +  # Create a scatter plot
  theme_minimal() +  # Use a minimal theme
  labs(title = "Scatter Plot of Random Data",
       x = "X Axis",
       y = "Y Axis")

print(demoPlot)
