from ctypes import *

import subprocess
import numpy as np
import os
import cv2
import time

os.system('go build -o puploc.so -buildmode=c-shared puploc.go')
pigo = cdll.LoadLibrary('./puploc.so')
os.system('rm puploc.so')

MAX_NDETS = 2024
ARRAY_DIM = 5
px, py = None, None

show_face = False

base_dir = "images"
source_imgs = ["sunglasses.png", "neon-yellow.png", "neon-green.png", "carnival.png", "carnival2.png", "neon-disco.png"]
img_idx = 0

# define class GoPixelSlice to map to:
# C type struct { void *data; GoInt len; GoInt cap; }
class GoPixelSlice(Structure):
	_fields_ = [
		("pixels", POINTER(c_ubyte)), ("len", c_longlong), ("cap", c_longlong),
	]

def rotateImage(image, angle):
  image_center = tuple(np.array(image.shape[1::-1]) / 2)
  rot_mat = cv2.getRotationMatrix2D(image_center, angle, 1.0)
  result = cv2.warpAffine(image, rot_mat, image.shape[1::-1], flags=cv2.INTER_LINEAR)

  return result

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

		# We have to consider the pupil pair added into the list.
		# That's why we are multiplying the detection length with 3.
		dets = list(res.reshape(-1, ARRAY_DIM))[0:dets_len*3]
		return dets

# initialize the camera
cap = cv2.VideoCapture(0)
cap.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
cap.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)

# Changing the camera resolution introduce a short delay in the camera initialization. 
# For this reason we should delay the object detection process with a few milliseconds.
time.sleep(0.4)

while(True):
	ret, frame = cap.read()
	pixs = np.ascontiguousarray(frame[:, :, 1].reshape((frame.shape[0], frame.shape[1])))
	pixs = pixs.flatten()

	# Verify if camera is intialized by checking if pixel array is not empty.
	if np.any(pixs):
		dets = process_frame(pixs) # pixs needs to be numpy.uint8 array

		if dets is not None:
			# We know that the detected faces are taking place in the first positions of the multidimensional array.
			for row, col, scale, q, angle in dets:			
				if q > 50:
					if angle == 0:
						px, py = col, row
					elif angle > 0:
						if show_face:
							cv2.rectangle(frame, (col-scale/2, row-scale/2), (col+scale/2, row+scale/2), (0, 0, 255), 2)

						src_img = cv2.imread(base_dir + "/" + source_imgs[img_idx], cv2.IMREAD_UNCHANGED)
						img_height, img_width, img_depth = src_img.shape

						if img_depth < 4:
							print("The provided image does not have an alpha channel.")
							exit(2)

						source_img = rotateImage(src_img, (angle-90))

						# Create the mask for the source image
						orig_mask = source_img[:,:,3]
						# Create the inverted mask for the source image
						orig_mask_inv = cv2.bitwise_not(orig_mask)
						# Convert the image to BGR
						source_img = source_img[:,:,:3]

						if scale < img_height or scale < img_width:				
							if img_height > img_width:		
								img_scale = float(scale)/float(img_height)
							else:
								img_scale = float(scale)/float(img_width)
						width, height = int(img_width*img_scale), int(img_height*img_scale)

						img = cv2.resize(source_img, (width, height), cv2.INTER_AREA)
						mask = cv2.resize(orig_mask, (width, height), cv2.INTER_AREA)
						mask_inv = cv2.resize(orig_mask_inv, (width, height), cv2.INTER_AREA)
					
						if px == None or py == None:
							continue
							
						y1 = row-scale/2+(row-scale/2-(py-height))
						y2 = row-scale/2+height+(row-scale/2-(py-height))
						x1 = col-scale/2
						x2 = col-scale/2+width
		
						if y1 < 0 or y2 < 0:
							continue
						roi = frame[y1:y2, x1:x2]
						roi_bg = cv2.bitwise_and(roi, roi, mask=mask_inv)
						roi_fg = cv2.bitwise_and(img, img, mask=mask)

						# join the roi_bg and roi_fg
						dst = cv2.add(roi_bg, roi_fg)
						frame[y1:y2, x1:x2] = dst			
						
	cv2.imshow('', frame)

	key = cv2.waitKey(1)
	if key & 0xFF == ord('q'):
		break
	elif key & 0xFF == ord('w'):
		show_face = not show_face	
	elif key & 0xFF == ord('e'):
		img_idx += 1
		if img_idx > len(source_imgs)-1:
			img_idx = 0
	elif key & 0xFF == ord('r'):
		img_idx -= 1
		if img_idx < 0:
			img_idx = len(source_imgs)-1

cap.release()
cv2.destroyAllWindows()