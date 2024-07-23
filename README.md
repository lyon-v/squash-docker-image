``squash-docker-image``
==================

Squash Docker OCI images and docker v2image to reduce the number of layers

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

- From source code

   ```
  go build -o squash-docker-image cmd/squash-docker-image/main.go
  
  cp squash-docker-image /usr/local/bin/
  ```

  

- To install this project, use the following command:

  ```
  go install github.com/lyon-v/squash-docker-image@latest
  ```

  


Usage
-----


    $squash-docker-image -h
    squash-docker-image is a CLI for squashing Docker images
    
    Usage:
      squash-docker-image [flags]
    
    Flags:
      -c, --cleanup              Remove source image from Docker after squashing
      -f, --from-layer string    Number of layers to squash or ID of the layer to squash from
      -h, --help                 help for squash-docker-image
      -i, --image string         Image to be squashed (required)
      -l, --load-image           Whether to load the image into Docker daemon after squashing (default true)
      -m, --message string       Specify a commit message for the new image (default "squash image")
      -o, --output-path string   Path where the image may be stored after squashing
      -t, --tag string           Specify the tag to be used for the new image
      -d, --tmp-dir string       Temporary directory to be created and used
      -v, --verbose              Verbose output
      -V, --version              Show version and exit



Examples
--------

    $ docker history  lyonv/ubuntubs:latest
    IMAGE          CREATED          CREATED BY                                      SIZE      COMMENT
    b936e0405275   3 minutes ago    /bin/bash                                       102MB     install git
    3ebdedc15512   5 minutes ago    /bin/bash                                       53.4MB    install net-tools
    0931cd6f5ea5   31 minutes ago   /bin/bash                                       97B       test
    0fe27b0007f9   37 minutes ago   /bin/bash                                       110MB     test
    5f5250218d28   3 weeks ago      /bin/sh -c #(nop)  CMD ["/bin/bash"]            0B        
    <missing>      3 weeks ago      /bin/sh -c #(nop) ADD file:e7cff353f027ecf0a…   72.8MB    
    <missing>      3 weeks ago      /bin/sh -c #(nop)  LABEL org.opencontainers.…   0B        
    <missing>      3 weeks ago      /bin/sh -c #(nop)  LABEL org.opencontainers.…   0B        
    <missing>      3 weeks ago      /bin/sh -c #(nop)  ARG LAUNCHPAD_BUILD_ARCH     0B        
    <missing>      3 weeks ago      /bin/sh -c #(nop)  ARG RELEASE                  0B



1.We want to squash last 3 layers from the ``imagedemo:v1`` image:



    $ squash-docker-image  -f 3 -t lyonv/ubuntubs:squashed -i lyonv/ubuntubs:latest
    time="2024-06-30 10:10:28" level=info msg="Running version 1.0.0" func=func1 file="main.go, line:69"
    time="2024-06-30 10:10:28" level=info msg="docker-squash version 1.0.0, Docker 25.0.2, API 1.44..." func=Run file="squash.go, line:97"
    time="2024-06-30 10:10:28" level=info msg="Squashing image: lyonv/ubuntubs:latest" func=Run file="squash.go, line:123"
    time="2024-06-30 10:10:28" level=info msg="Using /tmp/docker-squash-3440012490 as the temporary directory" func=prepareTmpDirectory file="OCIImage.go, line:1084"
    time="2024-06-30 10:10:28" level=info msg="Old image has 10 layers" func=beforeSquashing file="OCIImage.go, line:789"
    time="2024-06-30 10:10:28" level=info msg="Checking if squashing is necessary..." func=beforeSquashing file="OCIImage.go, line:816"
    time="2024-06-30 10:10:28" level=info msg="Attempting to squash last [ 3 ] layers..." func=beforeSquashing file="OCIImage.go, line:824"
    time="2024-06-30 10:10:28" level=info msg="Saving image sha256:b936e0405275c07d2856ea5d0864bfb9e2728e51cd4743dfc4b14610299ab016 to /tmp/docker-squash-3440012490/old directory..." func=saveImage file="OCIImage.go, line:964"
    time="2024-06-30 10:10:28" level=info msg="Try #1..." func=saveImage file="OCIImage.go, line:965"
    time="2024-06-30 10:10:28" level=info msg="Image saved successfully!" func=saveImage file="OCIImage.go, line:976"
    time="2024-06-30 10:10:28" level=info msg="Squashing image 'lyonv/ubuntubs:latest'..." func=beforeSquashing file="OCIImage.go, line:836"
    Starting squashing for /tmp/docker-squash-3440012490/new/squashed/layer.tar...
    Squashing file '/tmp/docker-squash-3440012490/old/blobs/sha256/d01de740e6f7a6123e6295090289786a47a00dd053962afd4971bcdbfd07fa37'...
    Squashing file '/tmp/docker-squash-3440012490/old/blobs/sha256/2e641c2f36b0cd022536a124ad07b48124620be457b090cd8afa50892b204cf8'...
    Squashing file '/tmp/docker-squash-3440012490/old/blobs/sha256/76faeca40935ebf05674db29e3a3b9b0ec9c1f0e4fb90f833b90a8986b44a4b4'...
    Squashing finishing!
    time="2024-06-30 10:10:28" level=info msg="Removing from disk already squashed layers..." func=afterSquashing file="OCIImage.go, line:69"
    time="2024-06-30 10:10:28" level=info msg="Cleaning up /tmp/docker-squash-3440012490/old temporary directory..." func=afterSquashing file="OCIImage.go, line:70"
    time="2024-06-30 10:10:28" level=info msg="Original image size: 327.350049 MB , Squashed image size: 326.528441 MB" func=afterSquashing file="OCIImage.go, line:81"
    Image size decreased by [ 0.25% ]
    Loading squashed image -->[ lyonv/ubuntubs:squashed ]...
    time="2024-06-30 10:10:28" level=info msg="Image loaded!" func=LoadSquashedImage file="OCIImage.go, line:1137"
    time="2024-06-30 10:10:28" level=info msg="Squashing complete" func=Run file="squash.go, line:134"
    Squashed image ID: [a61d40e17da5c87cad69cbae1d29d02ae9cdd2be58d156f74a48580ad2ca1f4f]

We can now confirm the layer structure:


    $ docker history lyonv/ubuntubs:squashed
    IMAGE          CREATED              CREATED BY                                      SIZE      COMMENT
    a61d40e17da5   About a minute ago                                                   155MB     squash image
    <missing>      39 minutes ago       /bin/bash                                       110MB     test
    <missing>      3 weeks ago          /bin/sh -c #(nop)  CMD ["/bin/bash"]            0B        
    <missing>      3 weeks ago          /bin/sh -c #(nop) ADD file:e7cff353f027ecf0a…   72.8MB    
    <missing>      3 weeks ago          /bin/sh -c #(nop)  LABEL org.opencontainers.…   0B        
    <missing>      3 weeks ago          /bin/sh -c #(nop)  LABEL org.opencontainers.…   0B        
    <missing>      3 weeks ago          /bin/sh -c #(nop)  ARG LAUNCHPAD_BUILD_ARCH     0B        
    <missing>      3 weeks ago          /bin/sh -c #(nop)  ARG RELEASE                  0B 





2.Let's squash all layers of the `imagedemo:v1` image into a single layer.:


    $ squash-docker-image -t lyonv/ubuntubs:squashed -i lyonv/ubuntubs:latest
    time="2024-06-30 10:10:31" level=info msg="Running version 1.0.0" func=func1 file="main.go, line:69"
    time="2024-06-30 10:10:31" level=info msg="docker-squash version 1.0.0, Docker 25.0.2, API 1.44..." func=Run file="squash.go, line:97"
    time="2024-06-30 10:10:31" level=info msg="Squashing image: lyonv/ubuntubs:latest" func=Run file="squash.go, line:123"
    time="2024-06-30 10:10:31" level=info msg="Using /tmp/docker-squash-1746182935 as the temporary directory" func=prepareTmpDirectory file="OCIImage.go, line:1084"
    time="2024-06-30 10:10:31" level=info msg="Old image has 10 layers" func=beforeSquashing file="OCIImage.go, line:789"
    time="2024-06-30 10:10:31" level=info msg="Checking if squashing is necessary..." func=beforeSquashing file="OCIImage.go, line:816"
    time="2024-06-30 10:10:31" level=info msg="Attempting to squash last [ 10 ] layers..." func=beforeSquashing file="OCIImage.go, line:824"
    time="2024-06-30 10:10:31" level=info msg="Saving image sha256:b936e0405275c07d2856ea5d0864bfb9e2728e51cd4743dfc4b14610299ab016 to /tmp/docker-squash-1746182935/old directory..." func=saveImage file="OCIImage.go, line:964"
    time="2024-06-30 10:10:31" level=info msg="Try #1..." func=saveImage file="OCIImage.go, line:965"
    time="2024-06-30 10:10:31" level=info msg="Image saved successfully!" func=saveImage file="OCIImage.go, line:976"
    time="2024-06-30 10:10:31" level=info msg="Squashing image 'lyonv/ubuntubs:latest'..." func=beforeSquashing file="OCIImage.go, line:836"
    Starting squashing for /tmp/docker-squash-1746182935/new/squashed/layer.tar...
    Squashing file '/tmp/docker-squash-1746182935/old/blobs/sha256/d01de740e6f7a6123e6295090289786a47a00dd053962afd4971bcdbfd07fa37'...
    Squashing file '/tmp/docker-squash-1746182935/old/blobs/sha256/2e641c2f36b0cd022536a124ad07b48124620be457b090cd8afa50892b204cf8'...
    Squashing file '/tmp/docker-squash-1746182935/old/blobs/sha256/76faeca40935ebf05674db29e3a3b9b0ec9c1f0e4fb90f833b90a8986b44a4b4'...
    Squashing file '/tmp/docker-squash-1746182935/old/blobs/sha256/a89290d197c69deb321cf62a95e7c22b8de1fa880f9431f38feebc76960a09f6'...
    Squashing file '/tmp/docker-squash-1746182935/old/blobs/sha256/3ec3ded77c0ce89e931f92aed086b2a2c774a6fbd51617853decc8afa4e1087a'...
    Squashing finishing!
    time="2024-06-30 10:10:31" level=info msg="Removing from disk already squashed layers..." func=afterSquashing file="OCIImage.go, line:69"
    time="2024-06-30 10:10:31" level=info msg="Cleaning up /tmp/docker-squash-1746182935/old temporary directory..." func=afterSquashing file="OCIImage.go, line:70"
    time="2024-06-30 10:10:31" level=info msg="Original image size: 327.350049 MB , Squashed image size: 325.765865 MB" func=afterSquashing file="OCIImage.go, line:81"
    Image size decreased by [ 0.48% ]
    Loading squashed image -->[ lyonv/ubuntubs:squashed ]...
    time="2024-06-30 10:10:31" level=info msg="Image loaded!" func=LoadSquashedImage file="OCIImage.go, line:1137"
    time="2024-06-30 10:10:31" level=info msg="Squashing complete" func=Run file="squash.go, line:134"
    Squashed image ID: [397ed2e1041e34cc4a211aa68f64983c90c320f598b8e3b1bdad6cecdafcdaf2]

Let's confirm the image st [opensource](..\opensource) ructure now:

    $ docker history lyonv/ubuntubs:squashed
    IMAGE          CREATED              CREATED BY   SIZE      COMMENT
    053dbe1293c8   About a minute ago                337MB     squash image


## TODO

- Compressing large images takes too long and needs optimization.
- Currently, image files support OCI; more formats need to be supported.



## Reference

- [docker-squash](https://github.com/goldmann/docker-squash) - written in python
- [docker-squash](https://github.com/jwilder/docker-squash) - written in go (not maintained)

