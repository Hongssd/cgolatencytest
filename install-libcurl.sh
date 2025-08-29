sudo apt update
sudo apt install build-essential libssl-dev zlib1g-dev libnghttp2-dev libpsl-dev
wget https://curl.se/download/curl-8.11.0.tar.gz
tar -xzvf curl-8.11.0.tar.gz
cd curl-8.11.0
./configure --with-openssl
make
sudo make install
sudo ldconfig