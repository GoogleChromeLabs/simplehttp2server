class Simplehttp2server < Formula
  desc "SimpleHTTP2Server"
  homepage "https://github.com/GoogleChrome/simplehttp2server"
  url "https://github.com/GoogleChrome/simplehttp2server/releases/download/3.1.0/simplehttp2server_darwin_amd64"
  sha256 "5129bb9a75bbb58eafa96432b5fd9902394b918b1b18981596264a776341ec08"
  version "3.1.0"

  def install
    system "chmod", "+x", "simplehttp2server_darwin_amd64"
    system "mkdir", "#{prefix}/bin"
    system "cp", "simplehttp2server_darwin_amd64", "#{prefix}/bin/simplehttp2server"
  end
end
