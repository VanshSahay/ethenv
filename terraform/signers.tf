locals {
  signers = [
    {
      address       = "ac756f0ccce30d4fe7b90164f9a6af2b185b5726"
      private_key   = "821fcb33010bcc092aba4b876e3a98a5f1d37942e19b485036c4f0bd253fef51"
      nodekey       = "117000a9c7c158fc6383bb31a493f4eabe503ed479e21116a6f96a6932b7b8db"
      nodekey_enode = "dd0ae3eaca2552e9274c6cf20e7236da793c0534fc9c8e3df253294caa11be0637c714136e9cf602f34d89fa36896d41ce9b0812d069e001048a54b32aeafa1d"
    },
    {
      address       = "98c798ab48397439c489e55e50cbbad5476d8307"
      private_key   = "b0b648a187109bcc06876a08fb5972f627cace71f7955ec6dcd24a97300e353b"
      nodekey       = "3e365473ae6725a512fc87edb3fb5f513cf70d1a054d38477e58e9977535eed6"
      nodekey_enode = "deb6afe458447335192e7f45463751377743af8afb4896cc93d6ba288c6375708ce6096780c1202cd6fa5de82b9625375d474bdfb01d346cf3a52394a3f206f2"
    },
    {
      address       = "9bbee5a8958e72f1cbc149ebf5a2e1bb92924e9a"
      private_key   = "86ad8b12c9dc0142e65084625292e43970d72901c6b77cc621f1ab779b87c1c7"
      nodekey       = "5bdb14992c8f41b181bc63d9a12fd168124bdce7cb33509d813925925a3ded89"
      nodekey_enode = "129c4fdb5282b1442a87c53ac1c9dd7015d73b1c0b5ed7fb7102268d693dea985c3547bd2cc3676199e9de418adf94236caea05e7bcfacfa2e50d589e6820719"
    },
  ]

  extradata = format(
    "0x%064d%s%0130d",
    0,
    join("", [for s in local.signers : s.address]),
    0,
  )

  alloc_entries = join(",\n", [
    for s in local.signers :
    "    \"0x${s.address}\": { \"balance\": \"1000000000000000000000000\" }"
  ])
}
