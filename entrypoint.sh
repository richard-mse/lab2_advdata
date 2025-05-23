#!/bin/sh
mkdir data
cd data
wget http://vmrum.isc.heia-fr.ch/dblpv14.json
mv dblpv14.json unsanitized.json
cd ..
go run .