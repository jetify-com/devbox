from pathlib import Path

import gdown
import gradio as gr
import numpy as np
import torch
from PIL import Image
from torch import nn

url = 'https://drive.google.com/uc?id=1dsk2JNZLRDjC-0J4wIQX_FcVurPaXaAZ'
output = 'pytorch_model.bin'
if not Path(output).exists():
    gdown.download(url, output, quiet=False)

LABELS = Path('class_names.txt').read_text().splitlines()

model = nn.Sequential(
    nn.Conv2d(1, 32, 3, padding='same'),
    nn.ReLU(),
    nn.MaxPool2d(2),
    nn.Conv2d(32, 64, 3, padding='same'),
    nn.ReLU(),
    nn.MaxPool2d(2),
    nn.Conv2d(64, 128, 3, padding='same'),
    nn.ReLU(),
    nn.MaxPool2d(2),
    nn.Flatten(),
    nn.Linear(1152, 256),
    nn.ReLU(),
    nn.Linear(256, len(LABELS)),
)
state_dict = torch.load(output, map_location='cpu')
model.load_state_dict(state_dict, strict=False)
model.eval()


def to_model_input(sketch):
    # The Sketchpad returns an RGBA canvas with dark strokes. The model was
    # trained on 28x28 grayscale drawings with light strokes on a dark
    # background, so convert the strokes to "ink" intensity and downscale.
    rgba = Image.fromarray(sketch["composite"]).convert("RGBA")
    gray = np.asarray(rgba.convert("L"), dtype=np.float32)
    alpha = np.asarray(rgba.getchannel("A"), dtype=np.float32) / 255.
    ink = Image.fromarray(((255. - gray) * alpha).astype(np.uint8))
    x = np.asarray(ink.resize((28, 28), Image.BOX), dtype=np.float32)
    if x.max() > 0:
        x = x / x.max()
    return torch.tensor(x).unsqueeze(0).unsqueeze(0)


def predict(sketch):
    if sketch is None or sketch.get("composite") is None:
        return None

    x = to_model_input(sketch)

    with torch.no_grad():
        out = model(x)

    probabilities = torch.nn.functional.softmax(out[0], dim=0)

    values, indices = torch.topk(probabilities, 5)

    return {LABELS[i]: v.item() for i, v in zip(indices, values)}


interface = gr.Interface(
    predict,
    inputs=gr.Sketchpad(label="Draw Here"),
    outputs=gr.Label(label="Guess"),
    live=True,
)

if __name__ == "__main__":
    interface.launch(footer_links=[])
