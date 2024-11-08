// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

contract Test {
    uint256[] public arr;
    address[] public addrArr;
    bytes[] public bytesArr;

    function setArr(uint256 len) public {
        for (uint256 i = 0; i < len; i++) {
            arr.push(i);
        }
    }

    function setAddrArr(uint256 len) public {
        for (uint256 i = 0; i < len; i++) {
            addrArr.push(address(uint160(i)));
        }
    }

    function setBytesArr(uint256 len) public {
        for (uint256 i = 0; i < len; i++) {
            bytesArr.push(abi.encodePacked(i));
        }
    }

    function setAllArrays(uint256 len) public {
        setArr(len);
        setAddrArr(len);
        setBytesArr(len);
    }
}