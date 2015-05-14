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


package xz

/*
#cgo LDFLAGS: -llzma
#include <string.h>
#include <stdlib.h>
#include <lzma.h>
*/
import "C"

import "errors"
import "io"
import "unsafe"
import "reflect"

type DecompressionReader struct {
    mBase          io.ReadCloser
    mStream        C.lzma_stream
    mReadBuffer    unsafe.Pointer
    mTempBuffer    unsafe.Pointer
    mTempBufferLen int
}
type CompressionWriter struct {
    mBase          io.WriteCloser
    mStream        C.lzma_stream
    mWriteBuffer   unsafe.Pointer
    mTempBuffer    unsafe.Pointer
    mTempBufferLen int
    mClosed        bool
}
const MEM_LIMIT = 256 * 1024 * 1024
var LZMA_OK = C.lzma_ret(0)
var LZMA_STREAM_END = C.lzma_ret(1)
var LZMA_NO_CHECK = C.lzma_ret(2)
var LZMA_UNSUPPORTED_CHECK = C.lzma_ret(3)
var LZMA_GET_CHECK = C.lzma_ret(4)
var LZMA_MEM_ERROR = C.lzma_ret(5)
var LZMA_MEMLIMIT_ERROR = C.lzma_ret(6)
var LZMA_FORMAT_ERROR = C.lzma_ret(7)
var LZMA_OPTIONS_ERROR = C.lzma_ret(8)
var LZMA_DATA_ERROR = C.lzma_ret(9)
var LZMA_BUF_ERROR = C.lzma_ret(10)
var LZMA_PROG_ERROR = C.lzma_ret(11)

var LZMA_RUN = C.lzma_action(0)
var LZMA_SYNC_FLUSH = C.lzma_action(1)
var LZMA_FULL_FLUSH = C.lzma_action(2)
var LZMA_FINISH = C.lzma_action(3)
var LZMA_FULL_BARRIER = C.lzma_action(4)

var LZMA_CHECK_NONE = C.lzma_check(0)
var LZMA_CHECK_CRC32 = C.lzma_check(1)
var LZMA_CHECK_CRC64 = C.lzma_check(4)
var LZMA_CHECK_SHA256 = C.lzma_check(10)

var IMPL_LZMA_BUFFER_LENGTH = C.size_t(4096)

type NopCloseReadWrapper struct {
    R io.Reader
}
func (r *NopCloseReadWrapper) Close () error {
    return nil
}
func (r *NopCloseReadWrapper) Read(data []byte) (int, error) {
    return r.R.Read(data)
}

type NopCloseWriteWrapper struct {
    W io.Writer
}
func (r *NopCloseWriteWrapper) Close () error {
    return nil
}
func (r *NopCloseWriteWrapper) Write(data []byte) (int, error) {
    return r.W.Write(data)
}


func NewDecompressionReader(r io.Reader) DecompressionReader {
    return NewDecompressionReadCloser(&NopCloseReadWrapper{r})
}
func NewDecompressionReadCloser(r io.ReadCloser) (retval DecompressionReader) {
    retval.mReadBuffer = C.malloc(IMPL_LZMA_BUFFER_LENGTH)
    retval.mTempBuffer = C.malloc(IMPL_LZMA_BUFFER_LENGTH)
    retval.mTempBufferLen = int(IMPL_LZMA_BUFFER_LENGTH)
    retval.mBase = r;
    //mStream = LZMA_STREAM_INIT;<-- assume we're zero initialized
    var ret C.lzma_ret
    ret = C.lzma_stream_decoder(
			&retval.mStream, MEM_LIMIT, 0);
	retval.mStream.avail_in = 0;
    if (ret != LZMA_OK) {
        switch(ret) {
          case LZMA_MEM_ERROR:
            panic("the stream decoder had insufficient memory");
          case LZMA_OPTIONS_ERROR:
            panic("the stream decoder had incorrect options for the system version");
          default:
            panic("the stream decoder was not initialized properly");
        }
    }
    return
};


func (dr *DecompressionReader) Read(data []byte) (int, error) {
    if len(data) > dr.mTempBufferLen {
        newLen := dr.mTempBufferLen * 3 / 2
        if newLen < len(data) {
            newLen = len(data)
        }
        C.free(dr.mTempBuffer)
        dr.mTempBufferLen = newLen
        dr.mTempBuffer = C.malloc(C.size_t(dr.mTempBufferLen))
    }
    tempSliceHdr := reflect.SliceHeader{
        Data:uintptr(dr.mTempBuffer),
        Len: len(data),
        Cap: len(data)}
    
    tempSlice := *(*[]byte)(unsafe.Pointer(&tempSliceHdr))

    readSliceHdr := reflect.SliceHeader{
        Data:uintptr(dr.mReadBuffer),
        Len: int(IMPL_LZMA_BUFFER_LENGTH),
        Cap: int(IMPL_LZMA_BUFFER_LENGTH)}
    
    readSlice := *(*[]byte)(unsafe.Pointer(&readSliceHdr))

    dr.mStream.next_out = (*C.uint8_t)(dr.mTempBuffer)
    dr.mStream.avail_out = C.size_t(len(data))
    for {
        var action C.lzma_action
        action = LZMA_RUN;
        var err error
        if (dr.mStream.avail_in == 0) {
            dr.mStream.next_in = (*C.uint8_t)(dr.mReadBuffer);
            var bytesRead int
            bytesRead, err = dr.mBase.Read(readSlice)
            dr.mStream.avail_in = C.size_t(bytesRead);
            if (bytesRead == 0) {
                action = LZMA_FINISH;
            }
        }
        var ret C.lzma_ret
        ret = C.lzma_code(&dr.mStream, action);
        if (dr.mStream.avail_out == 0 || ret == LZMA_STREAM_END) {
            writeSize := len(data) - int(dr.mStream.avail_out)
            copy(data[:writeSize], tempSlice[:writeSize])
            return writeSize, err
/////                                                (ret == LZMA_STREAM_END
/////                                                 || (ret == LZMA_OK &&writeSize > 0))
/////                                                 ? JpegError::nil() : err;
        }
        if (ret != LZMA_OK) {
            switch(ret) {
              case LZMA_FORMAT_ERROR:
                return 0, errors.New("Invalid XZ magic number")
              case LZMA_DATA_ERROR:
              case LZMA_BUF_ERROR:
                return len(data) - int(dr.mStream.avail_out), errors.New("Corrupt xz file")
              case LZMA_MEM_ERROR:
                panic("Memory allocation failed")
              default:
                panic("Unknown LZMA error code");
            }
        }
    }
    return 0, errors.New("Unreachable")
}

func (dr *DecompressionReader) Close() {
    C.lzma_end(&dr.mStream)
    C.free(dr.mReadBuffer)
    C.free(dr.mTempBuffer)
    dr.mReadBuffer = nil
    dr.mTempBuffer = nil
    dr.mTempBufferLen = 0
}


func NewCompressionWriter(w io.Writer) CompressionWriter {
    return NewCompressionWriteCloser(&NopCloseWriteWrapper{w})
}

func NewCompressionWriteCloser(w io.WriteCloser) (retval CompressionWriter) {
    retval.mWriteBuffer = C.malloc(IMPL_LZMA_BUFFER_LENGTH)
    retval.mTempBuffer = C.malloc(IMPL_LZMA_BUFFER_LENGTH)
    retval.mTempBufferLen = int(IMPL_LZMA_BUFFER_LENGTH)
    retval.mClosed = false;
    retval.mBase = w;
    //retval.mStream =  LZMA_STREAM_INIT;
    var ret C.lzma_ret
    ret = C.lzma_easy_encoder(&retval.mStream, 9, LZMA_CHECK_CRC64);
	retval.mStream.avail_in = 0;
    if (ret != LZMA_OK) {
        switch(ret) {
          case LZMA_MEM_ERROR:
            panic("the stream decoder had insufficient memory")
          case LZMA_OPTIONS_ERROR:
            panic("the stream decoder had incorrect options for the system version")
          case LZMA_UNSUPPORTED_CHECK:
            panic("Specified integrity check but not supported")
          default:
            panic("the stream decoder was not initialized properly")
        }
    }
    return
}

func (cw *CompressionWriter) Close() error {
    if cw.mClosed {
        panic("Closing a closed stream")
    }
    defer C.free(cw.mWriteBuffer)
    defer C.free(cw.mTempBuffer)
    defer cw.mBase.Close()
    cw.mClosed = true;
    for {
        var ret C.lzma_ret
        ret = C.lzma_code(&cw.mStream, LZMA_FINISH);
        if cw.mStream.avail_out == 0 || ret == LZMA_STREAM_END {
            writeSize := IMPL_LZMA_BUFFER_LENGTH - cw.mStream.avail_out;
            if writeSize > 0 {
                _, err := cw.mBase.Write(C.GoBytes(cw.mWriteBuffer, C.int(writeSize)));
                if err != nil {
                    return err;
                }
                cw.mStream.avail_out = IMPL_LZMA_BUFFER_LENGTH;
                cw.mStream.next_out = (*C.uint8_t)(cw.mWriteBuffer);
            }
        }
        if (ret == LZMA_STREAM_END) {
            return nil;
        }
    }

}

func (cw *CompressionWriter) Write(data []byte) (nWritten int, err error) {
    nWritten = 0
    err = nil
    cw.mStream.next_out = (*C.uint8_t)(cw.mWriteBuffer);
    cw.mStream.avail_out = IMPL_LZMA_BUFFER_LENGTH;
    if len(data) > cw.mTempBufferLen {
        newLen := cw.mTempBufferLen * 3 / 2
        if newLen < len(data) {
            newLen = len(data)
        }
        C.free(cw.mTempBuffer)
        cw.mTempBufferLen = newLen
        cw.mTempBuffer = C.malloc(C.size_t(cw.mTempBufferLen))
    }
    tempSliceHdr := reflect.SliceHeader{ Data:uintptr(cw.mTempBuffer),
                                      Len: len(data),
                                      Cap: len(data)}
    
    tempSlice := *(*[]byte)(unsafe.Pointer(&tempSliceHdr))
    copy(tempSlice, data)

    cw.mStream.next_in = (*C.uint8_t)(cw.mTempBuffer)
    cw.mStream.avail_in = C.size_t(len(data));

    for cw.mStream.avail_in > 0 {
        var ret C.lzma_ret
        ret = C.lzma_code(&cw.mStream, LZMA_RUN)
        if (cw.mStream.avail_in == 0 || cw.mStream.avail_out == 0 || ret == LZMA_STREAM_END) {
            writeSize := IMPL_LZMA_BUFFER_LENGTH - cw.mStream.avail_out
            if (writeSize > 0) {
                writeSliceHdr := reflect.SliceHeader{ Data:uintptr(cw.mWriteBuffer),
                                          Len: int(writeSize),
                                          Cap: int(writeSize)}
    
                writeSlice := *(*[]byte)(unsafe.Pointer(&writeSliceHdr))
                curNumWritten, curErr := cw.mBase.Write(writeSlice);
                cw.mStream.avail_out = IMPL_LZMA_BUFFER_LENGTH;
                cw.mStream.next_out = (*C.uint8_t)(cw.mWriteBuffer);
                nWritten += curNumWritten
                if (curErr != nil) {
                    err = curErr
                    return
                }
            }
        }
    }
    if err == nil {
        nWritten = len(data) // so as not to confuse the caller
    }
    return
}
