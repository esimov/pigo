from ctypes import *

import subprocess
import numpy as np
import os
import cv2
import time

os.system('go build -o blinkdet.so -buildmode=c-shared blinkdet.go')
pigo = cdll.LoadLibrary('./blinkdet.so')
os.system('rm blinkdet.so')

MAX_NDETS = 2024
ARRAY_DIM = 6
# Number of consecutive frames the eye must be below the threshold
EYE_CLOSED_CONSEC_FRAMES = 6

# define class GoPixelSlice to map to:
# C type struct { void *data; GoInt len; GoInt cap; }
class GoPixelSlice(Structure):
	_fields_ = [
		("pixels", POINTER(c_ubyte)), ("len", c_longlong), ("cap", c_longlong),
	]

# Obtain the camera pixels and transfer them to Go trough C types.
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
		res = np.ndarray(buffer=buffarr, dtype=c_longlong, shape=(MAX_NDETS, 5,))

		# The first value of the buffer aray represents the buffer length.
		dets_len = res[0][0]
		res = np.delete(res, 0, 0) # delete the first element from the array

		# We have to consider the pupil pair added into the list.
		# That's why we are multiplying the detection length with 3.
		dets = list(res.reshape(-1, 5))[0:dets_len*3]
		return dets

# initialize the camera
cap = cv2.VideoCapture(0)
cap.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
cap.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)

# Changing the camera resolution introduce a short delay in the camera initialization. 
# For this reason we should delay the object detection process with a few milliseconds.
time.sleep(0.4)

show_pupil = True
show_eyes = False
face_posy = 0
count_left, count_right = 0, 0

while(True):
	ret, frame = cap.read()
	pixs = np.ascontiguousarray(frame[:, :, 1].reshape((frame.shape[0], frame.shape[1])))
	pixs = pixs.flatten()

	# Verify if camera is intialized by checking if pixel array is not empty.
	if np.any(pixs):
		dets = process_frame(pixs) # pixs needs to be numpy.uint8 array

		if dets is not None:
			# We know that the detected faces are taking place in the first positions of the multidimensional array.
			for det in dets:
				if det[4] == 1: # 1 == face; 0 == pupil
					face_posy = det[1]
					cv2.rectangle(frame, 
							(int(det[1])-int(det[2]/2), int(det[0])-int(det[2]/2)), 
							(int(det[1])+int(det[2]/2), int(det[0])+int(det[2]/2)), 
							(0, 0, 255), 2
						)
				else:
					if show_pupil:
						count_left += 1
						count_right += 1

						x1, x2 = int(det[0])-int(det[2]*1.2), int(det[0])+int(det[2]*1.2)
						y1, y2 = int(det[1])-int(det[2]*1.2), int(det[1])+int(det[2]*1.2)
						subimg = frame[x1:x2, y1:y2]						
						
						if subimg is not None:
							gray = cv2.cvtColor(subimg, cv2.COLOR_BGR2GRAY)
							img_blur = cv2.medianBlur(gray, 1)

							if img_blur is not None:
								max_radius = int(det[2]*0.45)
								circles = cv2.HoughCircles(img_blur, cv2.HOUGH_GRADIENT, 1, int(det[2]*0.45), 
									param1=60, param2=21, minRadius=4, maxRadius=max_radius)
								
								if circles is not None:
									circles = np.uint16(np.around(circles))
									for i in circles[0, :]:
										if i[2] < max_radius and i[2] > 0:										
											# Draw outer circle																					
											cv2.circle(frame, (int(det[1]), int(det[0])), i[2], (0, 255, 0), 2)
											# Draw inner circle
											cv2.circle(frame, (int(det[1]), int(det[0])), 2, (255, 0, 255), 3)
								else:
									if face_posy < y1:
										count_left = 0
									else:
										count_right = 0								
						
						if count_left < EYE_CLOSED_CONSEC_FRAMES:						
							cv2.putText(frame, "Left blink!", (10, 30),
								cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 0, 255), 2)
						elif count_right < EYE_CLOSED_CONSEC_FRAMES:
							cv2.putText(frame, "Right blink!", (frame.shape[1]-150, 30),
								cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 0, 255), 2)
																							
						cv2.circle(frame, (int(det[1]), int(det[0])), 4, (0, 0, 255), -1, 8, 0)

					if show_eyes:
						cv2.rectangle(frame, 
							(int(det[1])-int(det[2]), int(det[0])-int(det[2])), 
							(int(det[1])+int(det[2]), int(det[0])+int(det[2])), 
							(0, 255, 0), 2
						)

	cv2.imshow('', frame)

	key = cv2.waitKey(1)
	if key & 0xFF == ord('q'):
		break
	elif key & 0xFF == ord('w'):
		show_pupil = not show_pupil
	elif key & 0xFF == ord('e'):
		show_eyes = not show_eyes

cap.release()
cv2.destroyAllWindows()