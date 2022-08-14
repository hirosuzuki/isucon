#!/bin/sh

# install required packages
sudo apt -y update
sudo apt -y install make gcc libmysqlclient-dev mysql-server apache2
sudo snap install --classic node
sudo snap install --classic go

# init database
cat webapp/config/database/isucon.sql | sudo mysql
gzip -dc webapp/config/database/isucon_db.sql.gz | sudo mysql

# build http_load
cd tools
tar xzf http_load/http_load-12mar2006.tar.gz
cd http_load-12mar2006
patch -p1 < ../http_load/http_load.patch
make
cd ../..

# build perl app
cd webapp/perl
curl -k -L http://cpanmin.us/ > ./cpanm
chmod +x ./cpanm
./cpanm -Lextlib -n --installdeps .
cd ../..

# build go app
cd webapp/go
go build -o isucon
cd ../..

# build benchmarcher
cd tools
../webapp/perl/cpanm -Lextlib -n JSON Furl
npm install
cd ..

# setup apache2
sudo a2enmod proxy
sudo a2enmod proxy_http
sudo a2dissite 000-default.conf
sudo cp setup/apache-default.conf /etc/apache2/sites-available/apache-default.conf
sudo a2ensite apache-default.conf
sudo systemctl reload apache2

# perl systemd setting
sed s/ubuntu/$USER/ setup/isucon.perl.service | sudo tee /etc/systemd/system/isucon.perl.service >/dev/null
sudo systemctl daemon-reload
sudo systemctl restart isucon.perl.service

# go systemd setting
sed s/ubuntu/$USER/ setup/isucon.go.service | sudo tee /etc/systemd/system/isucon.go.service >/dev/null
sudo systemctl daemon-reload

