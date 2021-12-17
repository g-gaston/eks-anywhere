#!/usr/bin/env bash


make e2e
# uncomment this to use embed config/bundles
#make eks-a-embed-config

# uncoment this to use --bundles-override through the e2e test functionality
# BUNDLES_OVERRIDE_PATH= ""
# export T_BUNDLES_OVERRIDE=true
# cp $BUNDLES_OVERRIDE_PATH bin/local-bundle-release.yaml"

./bin/e2e.test -test.v -test.run 'TestVerifyBundles'
