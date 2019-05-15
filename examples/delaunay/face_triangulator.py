from ctypes import *

import subprocess
import numpy as np
import os
import cv2
import time

os.system('go build -o pigo.so -buildmode=c-shared pigo.go')
pigo = cdll.LoadLibrary('./pigo.so')
os.system('rm pigo.so')

max_buff_len = 32000

# define class MapGoMethod to map to:
# C type struct { void *data; GoInt len; GoInt cap; }
class MapGoMethod(Structure):
	_fields_ = [
		("pixels", POINTER(c_ubyte)), ("len", c_longlong), ("cap", c_longlong),
	]

# Obtain the camera pixels and transfer them to Go trough Ctypes.
def process_frame(pixs):
	pixels = cast((c_ubyte * len(pixs))(*pixs), POINTER(c_ubyte))
	
	# call FindFaces
	faces = MapGoMethod(pixels, len(pixs), len(pixs))
	pigo.FindFaces.argtypes = [MapGoMethod]
	pigo.FindFaces.restype = c_void_p

	# Retrieve the pixel values from the Go function. 
	ndets = pigo.FindFaces(faces)
	data_pointer = cast(ndets, POINTER((c_longlong) * max_buff_len))
	
	if data_pointer :
		buffarr = ((c_longlong) * sizeof(data_pointer.contents)).from_address(addressof(data_pointer.contents))
		res = np.ndarray(buffer=buffarr, dtype=c_longlong, shape=(1, sizeof(data_pointer.contents),))
		res_flat = res.flatten()
		triangles = []

		for i in range(res_flat[0]):
			triangles.append(res_flat[2:res_flat[1]])
		return triangles

width, height = 640, 480

# initialize the camera
cap = cv2.VideoCapture(0)
cap.set(cv2.CAP_PROP_FRAME_WIDTH, width)
cap.set(cv2.CAP_PROP_FRAME_HEIGHT, height)

# Changing the camera resolution introduce a short delay in the camera initialization. 
# For this reason we should delay the object detection process with a few milliseconds.
time.sleep(0.4)

while(True):
	ret, frame = cap.read()
	bpixs = np.ascontiguousarray(frame[:, :, 0]).flatten() #Blue
	gpixs = np.ascontiguousarray(frame[:, :, 1]).flatten() #Green
	rpixs = np.ascontiguousarray(frame[:, :, 2]).flatten() #Red
	
	pixs = np.array(frame[:,:,::-1]).reshape(-1)

	# Verify if camera is intialized by checking if pixel array is not empty.
	if np.any(pixs):
		triangles = process_frame(pixs) # pixs needs to be np.uint8 array
		if triangles and triangles[0].any():
			for triangle in triangles:
				coords = np.array([triangle[1], triangle[0]])
				x, y = np.transpose(coords)
				if pixs.ndim > 1:
					continue
				rpxs = frame[:, :, 0][x:x+triangle[2], y:y+triangle[2]]
				gpxs = frame[:, :, 1][x:x+triangle[2], y:y+triangle[2]]
				bpxs = frame[:, :, 2][x:x+triangle[2], y:y+triangle[2]]

				tpxs = rpxs.flatten()
				btpxs = np.array(triangle[0:tpxs.size])
				gtpxs = np.array(triangle[tpxs.size:2*tpxs.size])
				rtpxs = np.array(triangle[2*tpxs.size:3*tpxs.size])

				for x in range(len(rpxs)):
					for y in range(len(rpxs[x])):
						bpxs[x,y] = btpxs[x + (y * triangle[2])]
						gpxs[x,y] = gtpxs[x + (y * triangle[2])]
						rpxs[x,y] = gtpxs[x + (y * triangle[2])]

	cv2.imshow('', frame)

	if cv2.waitKey(5) & 0xFF == ord('q'):
		break

cap.release()
cv2.destroyAllWindows()