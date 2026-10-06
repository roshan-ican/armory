import itertools
from pathlib import Path

import cv2

from embed import FaceError, distance, embed

PHOTOS = Path(__file__).parent / "photos"
EXTENSIONS = {".jpg", ".jpeg", ".png"}


def load():
    people = {}
    for folder in sorted(p for p in PHOTOS.iterdir() if p.is_dir()):
        vectors = []
        for path in sorted(folder.iterdir()):
            if path.suffix.lower() not in EXTENSIONS:
                continue
            image = cv2.imread(str(path))
            if image is None:
                print(f"{folder.name}/{path.name}: unreadable")
                continue
            try:
                vec, score = embed(image)
            except FaceError as e:
                print(f"{folder.name}/{path.name}: skipped ({e})")
                continue
            print(f"{folder.name}/{path.name}: ok, detector score {score:.2f}")
            vectors.append(vec)
        if vectors:
            people[folder.name] = vectors
    return people


def summary(values):
    return f"min {min(values):.2f}  avg {sum(values) / len(values):.2f}  max {max(values):.2f}"


def main():
    people = load()
    print()
    for name, vectors in people.items():
        pairs = [distance(a, b) for a, b in itertools.combinations(vectors, 2)]
        if pairs:
            print(f"same person  {name}: {summary(pairs)}")
    for a, b in itertools.combinations(people, 2):
        pairs = [distance(x, y) for x in people[a] for y in people[b]]
        print(f"different    {a} vs {b}: {summary(pairs)}")


main()
