class Simplehttp2server < Formula
  desc "SimpleHTTP2Server"
  homepage "https://github.com/GoogleChrome/simplehttp2server"
  url "https://github.com/GoogleChrome/simplehttp2server/releases/download/2.3.3/simplehttp2server_darwin_amd64"
  sha256 "10f6e3c4c60fbe431d585f9626874cf0dcf826c88bbf7e9681da6d7e4aaca9ae"

  def install
    system "chmod", "+x", "simplehttp2server_darwin_amd64"
    system "mkdir", "#{prefix}/bin"
    system "cp", "simplehttp2server_darwin_amd64", "#{prefix}/bin/simplehttp2server"
  end
end