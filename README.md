# Magnets Search Engine

Search engine for magnets from the Bittorrent's [DHT](https://en.wikipedia.org/wiki/Distributed_hash_table).

<img src="https://user-images.githubusercontent.com/45740/29029807-30f221be-7b89-11e7-8450-1dbcada38f1b.png" width="256px" alt=""/>

# Technologies

 * [Magnetico](https://github.com/boramalper/magnetico) for scrapping the DHT.
 * Elastic Search for the search database.
 * Go for the backend REST API.
 * HTML/CSS/JS for the frontend.
 * Docker, as always
 
# Why ?

This project is an alternative to `magneticow`, to play with Elastic Search and Go. The conclusion : I would have been faster using NodeJS and Elastic Search is kinda heavy, but it's still better than Python + Sqlite.
