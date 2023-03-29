---
title: Running Scripts
---

This doc describes how to configure and run scripts using `devbox run`. Scripts started with `devbox run` are launched in a interactive `devbox shell` that terminates once the script finishes, or is interrupted by CTRL-C. 

Scripts will run after your packages finish installing, and after your `init_hook` completes. 

## Configuring scripts

Scripts can be added in your `devbox.json`. Scripts require a unique name, and a command or list of commands to run: 

```json
"shell": {
    "init_hook": "echo \"Hello \"",
    "scripts": {
        "echo_once": "echo \"World\"", 
        "echo_twice": [
            "echo \"World\"",
            "echo \"Again\""
        ]
    }
}
```

## Running your scripts

To run a script, use `devbox run <script_name>`. This will start your shell, run your `init_hook`, and then run the script: 

```bash
$ devbox run echo_once
Installing nix packages. This may take a while... done.
Starting a devbox shell...
Hello
World

$ devbox run echo_twice
Installing nix packages. This may take a while... done.
Starting a devbox shell...
Hello
World
Again
```

Your devbox shell will exit once the last line of your script has finished running, or when you interrupt the script with CTRL-C (or a SIGINT signal).

## Running a One-off Command

You can use `devbox run` to run any command in your Devbox shell, even if you have not defined it as a script. For example, you can run the command below to print "Hello World" in your Devbox shell:

```bash
devbox run echo "Hello World"
```

For commands that use flags, you can use the `--` separator to tell Devbox that the rest of the command is a single command to run. For example, the following command will show you all the processes listening on port 80:

```bash
devbox run -- lsof -i :80
```


## Tips on using Scripts

1. Since `init_hook` runs everytime you start your shell, you should use primarily use it for setting environment variables and aliases. For longer running tasks like database setup, you can create and run a Devbox script
2. You can use Devbox scripts to start and manage long running background processes and daemons. 
   1. For example -- If you are working on a LAMP stack project, you can use scripts to start MySQL and Apache in separate shells and monitor their logs. Once you are done developing, you can use CTRL-C to exit the processes and shells
3. If a script feels too long to put it directly in `devbox.json`, you can save it as a shell script in your project, and then invoke it in your `devbox scripts`.
4. For more ideas, see the LAMP stack example in our [Devbox examples repo](https://github.com/jetpack-io/devbox/tree/main/examples/stacks/lapp-stack). 