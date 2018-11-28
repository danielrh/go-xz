package xz
import (
       "bytes"
       "io"
       "testing"
)

var dataToCompress = `
/*  libxz golang wrapper
 *
 *  Copyright (c) 2015, Daniel Reiter Horn
 *  All rights reserved.
 *
 *  Redistribution and use in source and binary forms, with or without
 *  modification, are permitted provided that the following conditions are
 *  met:
 *  * Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 *  * Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in
 *    the documentation and/or other materials provided with the
 *    distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS
 * IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED
 * TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A
 * PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER
 * OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
 * EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
 * PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
 * PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
 * LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
 * NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */
` + string([]byte{0,1,2,3,4,254,255})





func TestRoundTrip(t *testing.T) {
     initialReader :=  bytes.NewBufferString(dataToCompress) 
     var compressedData bytes.Buffer
     cw := NewCompressionWriter(&compressedData)
     for {
        var buffer [4096]byte
        nRead, readErr := initialReader.Read(buffer[:])
        if readErr != nil && readErr != io.EOF {
            panic(readErr)
        }
        nWrite, writeErr := cw.Write(buffer[:nRead])
        if writeErr != nil {
            panic(writeErr)
        }
        _ = nWrite
        if readErr != nil {
            break
        }
    }
    cw.Close()
    dr := NewDecompressionReader(bytes.NewBuffer(compressedData.Bytes()))
    var roundTrippedData bytes.Buffer
    for {
        var buffer [4096]byte
        nRead, readErr := dr.Read(buffer[:])
        if readErr != nil && readErr != io.EOF {
            panic(readErr)
        }
        nWrite, writeErr := roundTrippedData.Write(buffer[:nRead])
        if writeErr != nil || nWrite < nRead{
            panic(writeErr)
        }
        if readErr != nil {
            break
        }
    }
    dr.Close()
    if dataToCompress != string(roundTrippedData.Bytes()) {
        t.Errorf(dataToCompress + " != " + string(roundTrippedData.Bytes()))
    }
    if string(compressedData.Bytes()[1:5]) != "7zXZ" {
        t.Errorf("Invalid 7z signature")
    }
    if len(compressedData.Bytes()) > len(dataToCompress) {
        t.Errorf("Data to compress got bigger after xzing")
    }
}

type earlyEofReader struct {
    r io.Reader
}

func (eer *earlyEofReader) Read(p []byte) (int, error) {
    n, _ := eer.r.Read(p)
    return n, io.EOF
}

func TestRoundTripEarlyEof(t *testing.T) {
     byteArray := make([]byte, 8192)
     initialReader :=  bytes.NewBuffer(byteArray)
     var compressedData bytes.Buffer
     cw := NewCompressionWriter(&compressedData)
     for {
        var buffer [4096]byte
        nRead, readErr := initialReader.Read(buffer[:])
        if readErr != nil && readErr != io.EOF {
            panic(readErr)
        }
        nWrite, writeErr := cw.Write(buffer[:nRead])
        if writeErr != nil {
            panic(writeErr)
        }
        _ = nWrite
        if readErr != nil {
            break
        }
    }
    cw.Close()
    dr := NewDecompressionReader(&earlyEofReader{r: bytes.NewBuffer(compressedData.Bytes())})
    var roundTrippedData bytes.Buffer
    for {
        var buffer [4096]byte
        nRead, readErr := dr.Read(buffer[:])
        if readErr != nil && readErr != io.EOF {
            panic(readErr)
        }
        nWrite, writeErr := roundTrippedData.Write(buffer[:nRead])
        if writeErr != nil || nWrite < nRead{
            panic(writeErr)
        }
        if readErr != nil {
            break
        }
    }
    dr.Close()
    if !bytes.Equal(byteArray, roundTrippedData.Bytes()) {
        t.Errorf("Byte array does not match")
    }
    if string(compressedData.Bytes()[1:5]) != "7zXZ" {
        t.Errorf("Invalid 7z signature")
    }
    if len(compressedData.Bytes()) > len(byteArray) {
        t.Errorf("Data to compress got bigger after xzing")
    }
}
