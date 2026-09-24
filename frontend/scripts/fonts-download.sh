#!/usr/bin/env bash
set -e

#
# This script downloads our original font files from their source repos,
# checks their SHA-256 and puts them in our originalMedia folder.
#
# Satoshi is not downloaded here: it is not published on GitHub or Google Fonts
# and Fontshare has no versioned download link. Its variable woff2 is committed
# as delivered in src/assets/fonts, see src/assets/fonts/Satoshi-LICENSE.txt.
#

err_report() {
	echo "Error on line $(caller)" >&2
}

trap err_report ERR

ORIGINAL_FONTS_DIR="./originalMedia/fonts"

# JetBrains Mono v2.304, pinned to the commit of the release tag.
# GitHub publishes no SHA-256 for this release. The values below come from the
# first download, after checking that the git blob hashes of the files matched
# the ones GitHub lists for this commit:
#   fonts/variable/JetBrainsMono[wght].ttf  b60e77f5dbf5505436c1904cb7a9ac4111ee76d0
#   OFL.txt                                 8bee4148c1d54dbf5dae6d6c117fc80414266abb
# update these if there is a new version
JETBRAINS_MONO_COMMIT="cd5227bd1f61dff3bbd6c814ceaf7ffd95e947d9"
JETBRAINS_MONO_URL="https://raw.githubusercontent.com/JetBrains/JetBrainsMono/${JETBRAINS_MONO_COMMIT}"

# Downloads a file into the originalMedia folder and stops if its SHA-256 differs.
download_font_file() {
	URL=$1
	FILE_NAME=$2
	EXPECTED_SHA256=$3

	curl --fail --location --silent --show-error --output "${ORIGINAL_FONTS_DIR}/${FILE_NAME}" "$URL"
	echo "${EXPECTED_SHA256}  ${ORIGINAL_FONTS_DIR}/${FILE_NAME}" | sha256sum --check -
}

echo ""
echo "###################################################"
echo "# Download font files"
echo "###################################################"
echo ""

mkdir -p "$ORIGINAL_FONTS_DIR"

download_font_file \
	"${JETBRAINS_MONO_URL}/fonts/variable/JetBrainsMono%5Bwght%5D.ttf" \
	"JetBrainsMono[wght].ttf" \
	"662a196d58f1183bf2d77428b6d5283fe3f45161ab021bea4036bc98e5cac016"

download_font_file \
	"${JETBRAINS_MONO_URL}/OFL.txt" \
	"JetBrainsMono-OFL.txt" \
	"30f0c136e3c88e422d0791acd97238870f9054a9729bc34cf2ff0d4ed8cac4ad"

echo "Download complete"
