# Llama build and run

Simple Llama (generative AI) build and run with Devbox.

## Setup

- Make sure to have [devbox installed](https://www.jetify.com/devbox/docs/quickstart/#install-devbox)
- Clone this repo: `git clone https://github.com/jetify-com/devbox.git`
- `cd devbox/examples/data_science/llama/`
- `devbox shell`
- Once in devbox shell, there will be an available binary `llama` that you can use to run the built llama.cpp.
- `devbox run get_model`
- `devbox run llama`

## Updating the model

This example downloads [vicuna-7b model](https://huggingface.co/eachadea/ggml-vicuna-7b-1.1). You can change it to download another Llama model by editing the devbox.json

## Using Llama

`devbox run llama` runs the llama binary with a "hello world" prompt. To change that you can edit the prompt in devbox.json or once in devbox shell, run

```bash
llama -m ./models/vic7B/ggml-vic7b-q5_0.bin -n 512 -p "your custom prompt"
```

For more details on llama inference parameters refer to [llama.cpp docs](https://github.com/ggerganov/llama.cpp). Note that, instead of running `./main` you can run `llama` inside devbox shell.
