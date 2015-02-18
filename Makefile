# Copyright 2015 Dorival Pedroso & Raul Durand. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

all: gofem tools
.PHONY: gofem tools

gofem:
	go install

tools:
	(cd tools && make)
