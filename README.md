# cam-stream-recorder


Video de exemplo:
https://file-examples.com/storage/fe51b16b3a6848b0fa755ec/2017/04/file_example_MP4_1280_10MG.mp4

- usar o mediamtx como servidor de stream

- Usar o ffmpeg para gerar um stream de um video para o mediamtx.

ffmpeg -re -stream_loop -1 -i movie-example.mp4 -c copy -f rtsp rtsp://localhost:8554/mystream
