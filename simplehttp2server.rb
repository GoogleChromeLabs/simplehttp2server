class Simplehttp2server < Formula
  desc "SimpleHTTP2Server"
  homepage "https://github.com/GoogleChrome/simplehttp2server"
  url "https://github.com/GoogleChrome/simplehttp2server/releases/download/2.4.0/simplehttp2server_darwin_amd64"
  sha256 "e85b2551dd202566d130ae31fe0f8981c2e2d710389541763d2d088140af2ae4"
  version "2.4.0"

  def install
    system "chmod", "+x", "simplehttp2server_darwin_amd64"
    system "mkdir", "#{prefix}/bin"
    system "cp", "simplehttp2server_darwin_amd64", "#{prefix}/bin/simplehttp2server"
  end
end