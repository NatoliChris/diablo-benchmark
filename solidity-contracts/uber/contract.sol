// SPDX-License-Identifier: MIT
pragma solidity ^0.7.5;

contract ContractUber {
    uint public number_of_drivers = 10000;

    uint[] public driver_x;
    uint[] public driver_y;
    uint[] public client_x;
    uint[] public client_y;

    uint public maxlen = 10000;
    uint public y;


    function sqrt(uint x) public {
        uint z = (x + 1) / 2;
        y = x;
        while (z < y) {
            y = z;
            z = (x / z + z) / 2;
        }
    }

    event FindDriverEvent( uint driver_index);

    constructor() {
       for(uint i = 0; i < uint(number_of_drivers); i++) {
            driver_x.push((i*uint(sha256("i")))%maxlen);
            driver_y.push(((i+1)*uint(sha256("i")))%maxlen);
       }
       client_x.push((number_of_drivers*uint(sha256("i")))%maxlen);
       client_y.push(((number_of_drivers+1)*uint(sha256("i")))%maxlen);
    }

    function checkDistance() public {
        uint closer_driver;
        uint d;
        uint diff_x;
        uint diff_y;
	d = maxlen;

        for(uint i = 0; i < uint(number_of_drivers); i++) {
            diff_x = (client_x[0]-driver_x[i]) * (client_x[0]-driver_x[i]);
            diff_y = (client_y[1]-driver_y[i]) * (client_y[1]-driver_y[i]);
            sqrt(diff_x + diff_y);
            if (y < d) {
                d = y;
                closer_driver = i;
            } 
	}
        emit FindDriverEvent(closer_driver);
    }   
}
