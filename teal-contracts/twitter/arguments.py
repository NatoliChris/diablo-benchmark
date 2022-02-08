#!/usr/bin/env python3

import base64
import sys

if len(sys.argv) < 2:
    exit(1)

if sys.argv[1] == 'push()':
    print (base64.b64encode(b'Add').decode('ascii'))
    exit(0)

exit(1)
