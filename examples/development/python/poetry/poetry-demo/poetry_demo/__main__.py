from emoji import emojize


def main():
    """Prints a greeting message.

    This is a separate function so that it can be used as a script in the pyproject.toml
    """
    print(emojize(":rocket: Devbox with Poetry :rocket:"))


if __name__ == "__main__":
    main()

