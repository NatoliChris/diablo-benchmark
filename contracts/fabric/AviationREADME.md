# Aviation parts Online Marketplace

*Inspired by Hyperledger Fabric use case from Honeywell Aerospace* : https://www.hyperledger.org/learn/publications/honeywell-case-study

## Summary
<p>Since aviation is a heavily regulated industry, 
sales require certification from the U.S. Federal Aviation Administration 
and other agencies. Each part must be documented with a complete history
of its ownership, use, and repairs. 
Online buying of aircraft pieces in the style of Amazon
thus requires trust and we enable this 
 by having a trusted ledger with data integrity which lets us see :
	
   	- The entire lifecycle of a part and all its previous owners
   	- Anti counterfeit measures via certification and persistence
 </p>
 
 <p>Because aircraft pieces are expensive and large, deals are tend to be made 
 using purchase orders and not card payments. For users to be confident in their
 purchases, every purchase order is stored indefinitely on the ledger to
 provide tracability</p>
 
 ## Why benchmark it ?
 
 From the use case cited above :
  "Among the critical factors Honeywell needed were 
  low latency, high throughput, and fast send rates." As it is an Amazon type
  marketplace, thoses are legitimate requirements. 
 
 ## Implementation
 
 <p>We will assume every aircraft piece has a unique ID because the aviation world is 
 well regulated. An aircraft piece is defined by its id, description, 
 certification, value and *OWNER*. Sellers will post their offerings on the website 
 by adding the aircraft piece in the ledger world state using CreatePart().</p>
 
<p>
Every time a purchase is made using transferPart(), a purchase order is created and
placed on the ledger. Purchases are made using transferPart() because we want to
be able to follow the entire lifecycle of the part hence it is always transferred
from one owner to another.
</p>
 