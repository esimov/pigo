#!/usr/bin/env python2
import cv2
import imutils
from imutils.video import VideoStream
import time, sys

vs = VideoStream(resolution=(320, 240)).start()
time.sleep(1.0)
 
while(True):
    frame = vs.read()
    frame = imutils.resize(frame, width=640, height=480)
    
    #cv2.imshow('frame',frame)
    res = bytearray(cv2.imencode(".jpeg", frame)[1])
    size = str(len(res))

    sys.stdout.write("Content-Type: image/jpeg\r\n")
    sys.stdout.write("Content-Length: " + size + "\r\n\r\n")
    sys.stdout.write( res )
    sys.stdout.write("\r\n")
    sys.stdout.write("--informs\r\n")

    if cv2.waitKey(1) & 0xFF == ord('q'):
        break

cv2.destroyAllWindows()
vs.stop()