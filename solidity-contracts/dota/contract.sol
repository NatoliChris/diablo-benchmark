// SPDX-License-Identifier: MIT
pragma solidity ^0.7.5;
contract DecentralizedDota {
    int public number_of_players = 10;
    
    uint[] public team_positions_x;
    uint[] public team_positions_y;

    event NewStep(uint stepx, uint stepy);

    uint public maxlen = 250;

    constructor() {
       for(uint i =0; i < uint(number_of_players); i++){
	   team_positions_x.push(i**10%250);
           team_positions_y.push((i+1)**10%250);
       }
    }


    function update() public {

	for(uint i =0; i< uint(number_of_players); i++){
		team_positions_x[i] += 1;
		team_positions_y[i] += 1;

		if(team_positions_x[i] > maxlen){
   			team_positions_x[i]=0;
		}

		if (team_positions_x[i] < 0) {
			team_positions_x[i]=maxlen;
		} 

		if(team_positions_y[i] > maxlen){
   			team_positions_y[i]=0;
		}

		if (team_positions_y[i] < 0) {
			team_positions_y[i]=maxlen;
		}
	}	
    }
}
