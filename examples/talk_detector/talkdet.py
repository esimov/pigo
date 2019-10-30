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

# define class GoPixelSlice to map to:
# C type struct { void *data; GoInt len; GoInt cap; }
class GoPixelSlice(Structure):
	_fields_ = [
		("pixels", POINTER(c_ubyte)), ("len", c_longlong), ("cap", c_longlong),
	]

# Obtain the camera pixels and transfer them to Go trough Ctypes.
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

	if data_pointer :
		buffarr = ((c_longlong * ARRAY_DIM) * MAX_NDETS).from_address(addressof(data_pointer.contents))
		res = np.ndarray(buffer=buffarr, dtype=c_longlong, shape=(MAX_NDETS, ARRAY_DIM,))

		# The first value of the buffer aray represents the buffer length.
		dets_len = res[0][0]
		res = np.delete(res, 0, 0) # delete the first element from the array

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

showFaceDet = True
showPupil = True
showLandmarkPoints = True

while(True):
	ret, frame = cap.read()
	pixs = np.ascontiguousarray(frame[:, :, 1].reshape((frame.shape[0], frame.shape[1])))
	pixs = pixs.flatten()

	# Verify if camera is intialized by checking if pixel array is not empty.
	if np.any(pixs):
		dets = process_frame(pixs) # pixs needs to be numpy.uint8 array

		if dets is not None:
			# We know that the detected faces are taking place in the first positions of the multidimensional array.
			for row, col, scale, q, det_type, mouth_ar in dets:
				if q > 50:
					if det_type == 0: # 0 == face;
						if showFaceDet:
							cv2.rectangle(frame, (col-scale/2, row-scale/2), (col+scale/2, row+scale/2), (0, 0, 255), 2)
					elif det_type == 1: # 1 == pupil;
						if showPupil:
							cv2.circle(frame, (int(col), int(row)), 4, (0, 0, 255), -1, 8, 0)
					elif det_type == 2: # 2 == facial landmark;
						if showLandmarkPoints:
							cv2.circle(frame, (int(col), int(row)), 4, (0, 255, 0), -1, 8, 0)
					elif det_type == 3:
						if mouth_ar < MOUTH_AR_THRESH: # mouth is open
							cv2.putText(frame, "TALKING!", (10, 30),
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