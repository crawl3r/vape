# Vape  
  
One for the edgelords who blow clouds all day.  
Pass in a list of subdomains/domains/IPs/whatever and check whether they are part of a cloud providers range. The output will give us the URL, IP and the provider itself split by pipes. Should be easy enough to awk/sed your way to victory. If not, I'll add some more options.  
  
Currently checks for:  
* Akamai
* Cloudflare
* Incapsula
* Sucuri
  
I tend to check this when doing BB's due to false positives. I'm sure there are hundreds of reasons to continue the normal routine against cloud providers, low hanging fruit etc, but I tend to fine tune my flow based on the result.  

## Installing  
```
go get github.com/crawl3r/vape
```  
  
## Usage  
Standard Run  
```
cat urls.txt | ./vape
```
  
Run and save the output to file  
```
cat urls.txt | ./leakytap -o output.txt
```  
  
Run in quiet mode, only prints the identified cloud based IP addresses. 
```
cat urls.txt | ./leakytap -q
```
  
## License  
I'm just a simple skid. Licensing isn't a big issue to me, I post things that I find helpful online in the hope that others can:  
 A) learn from the code  
 B) find use with the code or   
 C) need to just have a laugh at something to make themselves feel better  
  
Either way, if this helped you - cool :)  
