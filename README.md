``squash-docker-image``
==================

Squash Docker OCI images to reduce the number of layers

The problem
-----------


There are many ways to build Docker images, such as using a Dockerfile or container-based builds. Regardless of the method, the resulting images typically contain many layers. Sometimes, these layers are neither necessary nor desired in the image. For example, the ADD instruction in a Dockerfile creates a layer containing specific files; or container-based builds may add a single layer to the image. Additionally, there may be temporary files within the image that, even if deleted in the next layer, Docker will still carry the unnecessary layer along with the image. Using container-based builds to create an image, then starting a new container from this image and performing package installations or deletions in the new container, and repeating this process, will also result in an increase in the number of layers. These situations waste time (more data to push, load, or save) and resources (leading to larger images).

Using squash-docker-image allows for effective management and reduction of image layers, preventing the proliferation of unnecessary layers.

Features
--------

- Allows compressing the image into a single layer
- Can squash from a selected layer to the end (not always possible, depends on the image)
- Supports Docker image v2 or OCI standard format images
- Squashed image can be reloaded into the Docker daemon or stored as a tar archive file





Installation
------------

From source code

    go build -o squash-docker-image cmd/squash-docker-image/main.go
    
    cp squash-docker-image /usr/local/bin/

To install this project, use the following command:

      ```sh
      go install github.com/lyon-v/squash-docker-image@latest


Usage
-----


    $ squash-docker-image -help
      Usage of squash-docker-image:
      -V    Show version and exit
      -c    Remove source image from Docker after squashing
      -cleanup
            Remove source image from Docker after squashing (shorthand: -c)
      -d string
            Temporary directory to be created and used
      -f string
            Number of layers to squash or ID of the layer to squash from
      -from-layer string
            Number of layers to squash or ID of the layer to squash from (shorthand: -f)
      -i string
            Image to be squashed (required)
      -image string
            Image to be squashed (required) (shorthand: -i)
      -l    Whether to load the image into Docker daemon after squashing (default true)
      -load-image
            Whether to load the image into Docker daemon after squashing (shorthand: -l) (default true)
      -m string
            Specify a commit message for the new image (default "squash image")
      -message string
            Specify a commit message for the new image (shorthand: -m) (default "squash image")
      -o string
            Path where the image may be stored after squashing
      -output-path string
            Path where the image may be stored after squashing (shorthand: -o)
      -t string
            Specify the tag to be used for the new image
      -tag string
            Specify the tag to be used for the new image (shorthand: -t)
      -tmp-dir string
            Temporary directory to be created and used (shorthand: -d)
      -v    Verbose output
      -verbose
            Verbose output (shorthand: -v)
      -version
            Show version and exit (shorthand: -V)



Examples
--------

    $ docker history imagedemo:v1
    IMAGE          CREATED        CREATED BY                                      SIZE      COMMENT
    475f688cd529   2 months ago   /usr/bin/supervisord -c /etc/supervisord.ini    1.5MB     build img
    <missing>      2 months ago   /usr/bin/supervisord -c /etc/supervisord.ini    1.5MB     build img
    <missing>      2 months ago   /usr/bin/supervisord -c /etc/supervisord.ini    1.49MB    build img
    <missing>      3 months ago                                                   481MB     shutdown button
    <missing>      4 months ago                                                   35.6MB    
    <missing>      9 months ago   /bin/sh -c #(nop)  ENTRYPOINT ["/bin/bash"]     0B        
    <missing>      9 months ago   /bin/sh -c #(nop) COPY file:0d00c7bf6116d565…   187B      
    <missing>      9 months ago   /bin/sh -c apt install tigervnc-standalone-s…   7.59MB    
    <missing>      9 months ago   /bin/bash                                       1.18GB    
    <missing>      2 years ago    /bin/sh -c #(nop)  CMD ["bash"]                 0B        
    <missing>      2 years ago    /bin/sh -c #(nop) ADD file:5d68d27cc15a80653…   72.8MB 



1.We want to squash last 3 layers from the ``imagedemo:v1`` image:



    $ squash-docker-image  --from-layer 3 --tag imagedemo:squashv1 --image imagedemo:v1
    time="2024-06-27 20:08:28" level=info msg="Running version 1.0.0" func=run file="cli.go, line:66"
    time="2024-06-27 20:08:28" level=info msg="docker-squash version 1.0.0, Docker 25.0.2, API 1.44..." func=Run file="squash.go, line:101"
    time="2024-06-27 20:08:28" level=info msg="Squashing image: imagedemo:v1" func=Run file="squash.go, line:127"
    time="2024-06-27 20:08:28" level=info msg="Using /tmp/docker-squash-364890302 as the temporary directory" func=prepareTmpDirectory file="OCIImage.go, line:1084"
    time="2024-06-27 20:08:28" level=info msg="Old image has 11 layers" func=beforeSquashing file="OCIImage.go, line:789"
    time="2024-06-27 20:08:28" level=info msg="Checking if squashing is necessary..." func=beforeSquashing file="OCIImage.go, line:816"
    time="2024-06-27 20:08:28" level=info msg="Attempting to squash last [ 3 ] layers..." func=beforeSquashing file="OCIImage.go, line:824"
    time="2024-06-27 20:08:28" level=info msg="Saving image sha256:475f688cd52948ea247181b19bdb2e0f88cd9e9ec5cf60f7752c253fac892241 to /tmp/docker-squash-364890302/old directory..." func=saveImage file="OCIImage.go, line:964"
    time="2024-06-27 20:08:28" level=info msg="Try #1..." func=saveImage file="OCIImage.go, line:965"
    time="2024-06-27 20:08:28" level=info msg="Image saved successfully!" func=saveImage file="OCIImage.go, line:976"
    time="2024-06-27 20:08:28" level=info msg="Squashing image 'imagedemo:v1'..." func=beforeSquashing file="OCIImage.go, line:836"
    Starting squashing for /tmp/docker-squash-364890302/new/squashed/layer.tar...
    Squashing file '/tmp/docker-squash-364890302/old/blobs/sha256/e5decb59dbcfbc37c1ce7b12f2977981ac4be4009cea1b4506e7c4261c030570'...
    Squashing file '/tmp/docker-squash-364890302/old/blobs/sha256/4688581830b3564d8ef0a441d8824e01fb18cdc6f3a642333a50a40436fe1d46'...
    Squashing file '/tmp/docker-squash-364890302/old/blobs/sha256/2d59ac497c759308f3d11131616ee5b000309760f56696215370f1756b555c4b'...
    Squashing finishing!
    time="2024-06-27 20:08:28" level=info msg="Removing from disk already squashed layers..." func=afterSquashing file="OCIImage.go, line:69"
    time="2024-06-27 20:08:28" level=info msg="Cleaning up /tmp/docker-squash-364890302/old temporary directory..." func=afterSquashing file="OCIImage.go, line:70"
    time="2024-06-27 20:08:28" level=info msg="Original image size: 1746.303209 MB , Squashed image size: 1743.394833 MB" func=afterSquashing file="OCIImage.go, line:81"
    Image size decreased by [ 0.17% ]
    Loading squashed image -->[ imagedemo:squashv1 ]...
    time="2024-06-27 20:08:29" level=info msg="Image loaded!" func=LoadSquashedImage file="OCIImage.go, line:1137"
    time="2024-06-27 20:08:29" level=info msg="Squashing complete" func=Run file="squash.go, line:138"
    suqashed imageId: [dd18dc20272e3f37d370a7fe14d7cb467013e3f0660659da986cf6e780983e28] 

We can now confirm the layer structure:


    $ docker history imagedemo:squashv1 
    IMAGE          CREATED              CREATED BY                                      SIZE      COMMENT
    dd18dc20272e   About a minute ago                                                   1.51MB    squash image
    <missing>      3 months ago                                                         481MB     shutdown button
    <missing>      4 months ago                                                         35.6MB    
    <missing>      9 months ago         /bin/sh -c #(nop)  ENTRYPOINT ["/bin/bash"]     0B        
    <missing>      9 months ago         /bin/sh -c #(nop) COPY file:0d00c7bf6116d565…   187B      
    <missing>      9 months ago         /bin/sh -c apt install tigervnc-standalone-s…   7.59MB    
    <missing>      9 months ago         /bin/bash                                       1.18GB    
    <missing>      2 years ago          /bin/sh -c #(nop)  CMD ["bash"]                 0B        
    <missing>      2 years ago          /bin/sh -c #(nop) ADD file:5d68d27cc15a80653…   72.8MB   





2.Let's squash all layers of the `imagedemo:v1` image into a single layer.:


    $ squash-docker-image -tag imagedemo:squashv1 -image imagedemo:v1
    time="2024-06-27 21:09:07" level=info msg="Running version 1.0.0" func=run file="cli.go, line:66"
    time="2024-06-27 21:09:07" level=info msg="docker-squash version 1.0.0, Docker 25.0.2, API 1.44..." func=Run file="squash.go, line:101"
    time="2024-06-27 21:09:07" level=info msg="Squashing image: imagedemo:v1" func=Run file="squash.go, line:127"
    time="2024-06-27 21:09:07" level=info msg="Using /tmp/docker-squash-2319696826 as the temporary directory" func=prepareTmpDirectory file="OCIImage.go, line:1084"
    time="2024-06-27 21:09:07" level=info msg="Old image has 11 layers" func=beforeSquashing file="OCIImage.go, line:789"
    time="2024-06-27 21:09:07" level=info msg="Checking if squashing is necessary..." func=beforeSquashing file="OCIImage.go, line:816"
    time="2024-06-27 21:09:07" level=info msg="Attempting to squash last [ 11 ] layers..." func=beforeSquashing file="OCIImage.go, line:824"
    time="2024-06-27 21:09:07" level=info msg="Saving image sha256:475f688cd52948ea247181b19bdb2e0f88cd9e9ec5cf60f7752c253fac892241 to /tmp/docker-squash-2319696826/old directory..." func=saveImage file="OCIImage.go, line:964"
    time="2024-06-27 21:09:07" level=info msg="Try #1..." func=saveImage file="OCIImage.go, line:965"
    time="2024-06-27 21:09:07" level=info msg="Image saved successfully!" func=saveImage file="OCIImage.go, line:976"
    time="2024-06-27 21:09:07" level=info msg="Squashing image 'imagedemo:v1'..." func=beforeSquashing file="OCIImage.go, line:836"
    Starting squashing for /tmp/docker-squash-2319696826/new/squashed/layer.tar...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/e5decb59dbcfbc37c1ce7b12f2977981ac4be4009cea1b4506e7c4261c030570'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/4688581830b3564d8ef0a441d8824e01fb18cdc6f3a642333a50a40436fe1d46'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/2d59ac497c759308f3d11131616ee5b000309760f56696215370f1756b555c4b'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/070d5f06678fa03bfd62ea0bdcd7cebfaaeeb99a1ec99598d870c33ebd27148f'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/6931773e145dbb5d33e707f87ddd7054683a274a8a1eb3b12313a1c3d92fe91f'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/bf7ddd2cfb36ffc3e23484a2ef1264b5aae3f69da8a564e041e102c6ab36bdd0'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/2adc4752e02732e8fc9cb6e73a21866630209b426fed5bff7681891a3e608460'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/47c55dc040c4924e457f895524ebe6fbe1b7d07309869b20b81faf7cc1338f08'...
    Squashing file '/tmp/docker-squash-2319696826/old/blobs/sha256/9f54eef412758095c8079ac465d494a2872e02e90bf1fb5f12a1641c0d1bb78b'...
    Squashing finishing!
    time="2024-06-27 21:09:07" level=info msg="Removing from disk already squashed layers..." func=afterSquashing file="OCIImage.go, line:69"
    time="2024-06-27 21:09:07" level=info msg="Cleaning up /tmp/docker-squash-2319696826/old temporary directory..." func=afterSquashing file="OCIImage.go, line:70"
    time="2024-06-27 21:09:07" level=info msg="Original image size: 1746.303209 MB , Squashed image size: 1635.164439 MB" func=afterSquashing file="OCIImage.go, line:81"
    Image size decreased by [ 6.36% ]
    Loading squashed image -->[ imagedemo:squashv1 ]...
    time="2024-06-27 21:09:07" level=info msg="Image loaded!" func=LoadSquashedImage file="OCIImage.go, line:1137"
    time="2024-06-27 21:09:07" level=info msg="Squashing complete" func=Run file="squash.go, line:138"
    suqashed imageId: [22c61cd502504ea7c7f1d19dbcdddb4205124ededdcb9c15e0841b27a455e28e]

Let's confirm the image st [opensource](..\opensource) ructure now:

    $ docker history imagedemo:squashv1
    IMAGE          CREATED              CREATED BY   SIZE      COMMENT
    22c61cd50250   About a minute ago                1.66GB    squash image


## TODO

- Compressing large images takes too long and needs optimization.
- Currently, image files support OCI; more formats need to be supported.



## Reference

- [docker-squash](https://github.com/goldmann/docker-squash) - written in python
- [docker-squash](https://github.com/jwilder/docker-squash) - written in go (not maintained)

