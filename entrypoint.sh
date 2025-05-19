#!/bin/sh
mkdir data
cd data
wget http://vmrum.isc.heia-fr.ch/dblpv13.json
mv test.json unsanitized.json
cd ..
go run .