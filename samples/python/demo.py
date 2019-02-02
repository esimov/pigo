from ctypes import *

import subprocess
import numpy
import os
import cv2
import time

os.system('go build -o pigo.so -buildmode=c-shared pigo.go')
pigo = cdll.LoadLibrary('./pigo.so')
os.system('rm pigo.so')

MAX_NDETS = 2048

# define class GoPixelSlice to map to:
# C type struct { void *data; GoInt len; GoInt cap; }
class GoPixelSlice(Structure):
	_fields_ = [
		("pixels", POINTER(c_ubyte)), ("len", c_longlong), ("cap", c_longlong),
	]

def process_frame(pixs):
	dets = numpy.zeros(3*MAX_NDETS, dtype=numpy.float32)
	pixs = pixs.flatten()
	pixels = cast((c_ubyte * len(pixs))(*pixs), POINTER(c_ubyte))
	
	# call FindFaces
	faces = GoPixelSlice(pixels, len(pixs), len(pixs))
	pigo.FindFaces.argtypes = [GoPixelSlice]
	pigo.FindFaces.restype = c_void_p
	#pigo.FindFaces.restype = POINTER((c_longlong * 3) * MAX_NDETS)
	#pigo.FindFaces.restype = numpy.ctypeslib.ndpointer(dtype = c_longlong, shape = (MAX_NDETS, 3, ))

	ndets = pigo.FindFaces(faces)
	data_pointer = cast(ndets, POINTER((c_longlong * 3) * MAX_NDETS))
	
	if data_pointer :
		#print(data_pointer.contents)
		#addr = addressof(data_pointer.contents)
		#new_array =  cast(addr, POINTER((c_longlong * 3) * MAX_NDETS)).contents
		new_array = ((c_longlong * 3) * MAX_NDETS).from_address(addressof(data_pointer.contents))
		# new_array = numpy.ctypeslib.as_array(data_pointer,shape=(8192,))

		# buffer = numpy.core.multiarray.int_asbuffer(addressof(new_array), 3*MAX_NDETS)
		# a = numpy.frombuffer(buffer, int)
		# print(a)

		res = numpy.ndarray(buffer=new_array, dtype=c_longlong, shape=(MAX_NDETS, 3,))
		dets_len = res[0][0]
		print(dets_len)
		res = numpy.delete(res, 0, 0)
		#print(res)
		dets = numpy.frombuffer(data_pointer.contents, dtype=numpy.dtype(int))
		dets = numpy.trim_zeros(dets)
		#dets = dets.astype(int)
		#print(dets)

		dets = list(res.reshape(-1, 3))[0:dets_len]

		return dets
		#print(dets)
	#print(dets)
	#return list(dets.reshape(-1, 4))[0:dets]
	
	#res = pigo.FindFaces(pixels, gray.shape[0], gray.shape[1], cascade)
	#print(res.dets)

cap = cv2.VideoCapture(0)
cap.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
cap.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)

w = cap.get(cv2.CAP_PROP_FRAME_WIDTH)
h = cap.get(cv2.CAP_PROP_FRAME_HEIGHT)
print (w,h)

while(True):
	ret, frame = cap.read()
	pixs = numpy.ascontiguousarray(frame[:, :, 1].reshape((frame.shape[0], frame.shape[1])))

	if not numpy.any(pixs):
		continue

	dets = process_frame(pixs) # pixs needs to be numpy.uint8 array
	print(dets)
	if dets is not None:
		for det in dets:
			cv2.circle(frame, (int(det[1]), int(det[0])), int(det[2]/2.0), (0, 0, 255), 2)

	cv2.imshow('', frame)
	
	if cv2.waitKey(5) & 0xFF == ord('q'):
		break

cap.release()
cv2.destroyAllWindows()