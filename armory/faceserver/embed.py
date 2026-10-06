from pathlib import Path

import cv2
import numpy as np

BASE = Path(__file__).parent
DETECT_MODEL = str(BASE / "models" / "face_detection_yunet_2023mar.onnx")
RECOGNIZE_MODEL = str(BASE / "models" / "face_recognition_sface_2021dec.onnx")
MIN_SCORE = 0.8

detector = cv2.FaceDetectorYN.create(DETECT_MODEL, "", (320, 320), MIN_SCORE, 0.3, 5000)
recognizer = cv2.FaceRecognizerSF.create(RECOGNIZE_MODEL, "")


class FaceError(Exception):
    pass


def embed(image):
    h, w = image.shape[:2]
    detector.setInputSize((w, h))
    _, faces = detector.detect(image)
    if faces is None or len(faces) == 0:
        raise FaceError("no face")
    if len(faces) > 1:
        raise FaceError("more than one face")
    face = faces[0]
    aligned = recognizer.alignCrop(image, face)
    vec = recognizer.feature(aligned).flatten()
    vec = vec / np.linalg.norm(vec)
    return vec, float(face[-1])


def distance(a, b):
    return float(np.linalg.norm(a - b))
