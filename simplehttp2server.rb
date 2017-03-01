class Simplehttp2server < Formula
  desc "SimpleHTTP2Server"
  homepage "https://github.com/GoogleChrome/simplehttp2server"
  url "https://github.com/GoogleChrome/simplehttp2server/releases/download/3.1.0/simplehttp2server_darwin_amd64"
  sha256 "eedee5e90ae25332b8cd9bc61f68ee763b30e2e0e2a498c7245b916864da745c"
  version "3.1.1"

  def install
    system "chmod", "+x", "simplehttp2server_darwin_amd64"
    system "mkdir", "#{prefix}/bin"
    system "cp", "simplehttp2server_darwin_amd64", "#{prefix}/bin/simplehttp2server"
  end
end
