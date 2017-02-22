class Simplehttp2server < Formula
  desc "SimpleHTTP2Server"
  homepage "https://github.com/GoogleChrome/simplehttp2server"
  url "https://github.com/GoogleChrome/simplehttp2server/releases/download/3.0.1/simplehttp2server_darwin_amd64"
  sha256 "3a090db3f9f52ec240329ff08ec9fe5e7b07b41c78d081eb8e46036360d4ce67"
  version "3.0.1"

  def install
    system "chmod", "+x", "simplehttp2server_darwin_amd64"
    system "mkdir", "#{prefix}/bin"
    system "cp", "simplehttp2server_darwin_amd64", "#{prefix}/bin/simplehttp2server"
  end
end
