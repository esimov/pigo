from ctypes import *

import subprocess
import numpy as np
import os
import cv2
import time

os.system('go build -o talkdet.so -buildmode=c-shared talkdet.go')
pigo = cdll.LoadLibrary('./talkdet.so')
os.system('rm talkdet.so')

MAX_NDETS = 2024
ARRAY_DIM = 6

MOUTH_AR_THRESH = 0.2
MOUTH_AR_CONSEC_FRAMES = 5

def verify_alpha_channel(frame):
    try:
        frame.shape[3]  # 4th position
    except IndexError:
        frame = cv2.cvtColor(frame, cv2.COLOR_BGR2BGRA)
    return frame

def alpha_blend(frame_1, frame_2, mask):
    alpha = mask/255.0
    blended = cv2.convertScaleAbs(frame_1*(1-alpha) + frame_2*alpha)
    return blended

def apply_circle_focus_blur(frame, x, y, dim, blur):
    frame = verify_alpha_channel(frame)
    height, width, _ = frame.shape
    mask = np.zeros((height, width, 4), dtype='uint8')
    cv2.circle(mask, (int(x), int(y)), int(dim/1.5),
               (255, 255, 255), -1, cv2.LINE_AA)
    mask = cv2.blur(mask, (blur, blur), cv2.BORDER_DEFAULT)
    blured = cv2.blur(frame, (blur, blur), cv2.BORDER_DEFAULT)
    blended = alpha_blend(frame, blured, 255-mask)
    frame = cv2.cvtColor(blended, cv2.COLOR_BGRA2BGR)
    return frame

# define class GoPixelSlice to map to:
# C type struct { void *data; GoInt len; GoInt cap; }
class GoPixelSlice(Structure):
    _fields_ = [
        ("pixels", POINTER(c_ubyte)), ("len", c_longlong), ("cap", c_longlong),
    ]

# Obtain the camera pixels and transfer them to Go trough Ctypes
def process_frame(pixs):
    dets = np.zeros(ARRAY_DIM * MAX_NDETS, dtype=np.float32)
    pixels = cast((c_ubyte * len(pixs))(*pixs), POINTER(c_ubyte))

    # call FindFaces
    faces = GoPixelSlice(pixels, len(pixs), len(pixs))
    pigo.FindFaces.argtypes = [GoPixelSlice]
    pigo.FindFaces.restype = c_void_p

    # Call the exported FindFaces function from Go.
    ndets = pigo.FindFaces(faces)
    data_pointer = cast(ndets, POINTER((c_longlong * ARRAY_DIM) * MAX_NDETS))

    if data_pointer:
        buffarr = ((c_longlong * ARRAY_DIM) *
                   MAX_NDETS).from_address(addressof(data_pointer.contents))
        res = np.ndarray(buffer=buffarr, dtype=c_longlong,
                         shape=(MAX_NDETS, ARRAY_DIM,))

        # The first value of the buffer aray represents the buffer length.
        dets_len = res[0][0]
        res = np.delete(res, 0, 0)  # delete the first element from the array

        # We have to multiply the detection length with the total
        # detection points(face, pupils and facial lendmark points), in total 18
        dets = list(res.reshape(-1, ARRAY_DIM))[0:dets_len*19]
        return dets

# initialize the camera
cap = cv2.VideoCapture(0)
cap.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
cap.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)

# Changing the camera resolution introduce a short delay in the camera initialization.
# For this reason we should delay the object detection process with a few milliseconds.
time.sleep(0.4)

showFaceDet = False
showPupil = True
showLandmarkPoints = True

counter = 0
talking = False

while(True):
    ret, frame = cap.read()
    pixs = np.ascontiguousarray(
        frame[:, :, 1].reshape((frame.shape[0], frame.shape[1])))
    pixs = pixs.flatten()

    # Verify if camera is intialized by checking if pixel array is not empty.
    if np.any(pixs):
        dets = process_frame(pixs)  # pixs needs to be numpy.uint8 array

        if dets is not None:
            face_col, face_row, face_dim = 0, 0, 0
            # We know that the detected faces are taking place in the first positions of the multidimensional array.
            for row, col, scale, q, det_type, mouth_ar in dets:
                if q > 50:
                    if det_type == 0:  # 0 == face;
                        face_col, face_row, face_dim = col, row, scale
                        if showFaceDet:
                            cv2.rectangle(frame, (col-scale/2, row-scale/2), (col+scale/2, row+scale/2), (0, 0, 255), 2)
                    elif det_type == 1:  # 1 == pupil;
                        if showPupil:
                            cv2.circle(frame, (int(col), int(row)), 4, (0, 0, 255), -1, 8, 0)
                    elif det_type == 2:  # 2 == facial landmark;
                        if showLandmarkPoints:
                            cv2.circle(frame, (int(col), int(row)), 4, (0, 255, 0), -1, 8, 0)
                    elif det_type == 3:
                        if mouth_ar < MOUTH_AR_THRESH: # mouth is open
                            talking = True
                            counter = 0
                        else: # mouth is closed
                            if counter < MOUTH_AR_CONSEC_FRAMES:
                                counter += 1
                            else:
                                talking = False
                                counter = 0

                        if talking and counter < MOUTH_AR_CONSEC_FRAMES:
                            frame = apply_circle_focus_blur(frame, face_col, face_row, face_dim, 25)
                            cv2.putText(frame, "Bla bla bla...", (10, 30),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 0, 255), 2)

    cv2.imshow('', frame)

    key = cv2.waitKey(1)
    if key & 0xFF == ord('q'):
        break
    elif key & 0xFF == ord('w'):
        showFaceDet = not showFaceDet
    elif key & 0xFF == ord('e'):
        showPupil = not showPupil
    elif key & 0xFF == ord('r'):
        showLandmarkPoints = not showLandmarkPoints

cap.release()
cv2.destroyAllWindows()