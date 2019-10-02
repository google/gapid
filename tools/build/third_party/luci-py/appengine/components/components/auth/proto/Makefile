compile:
	# If we simply compile *.proto files relative to this dir, the recorded file
	# names in the generated _pb2.py are relative to this dir too which may
	# introduce naming collisions in the protobuf registry.
	#
	# Move the root of proto compilation to //appengine/components to "namespace"
	# proto files. This is consistent with other components assuming that
	# "components" is in the import path.
	cd ../../../ && tools/compile_proto.py --proto_path=. components/auth/proto
