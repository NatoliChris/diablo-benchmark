script {
  use Owner::Content;

  fun main(a: signer, data: vector<u8>) {
    Content::publish(&a, data);
  }
}
