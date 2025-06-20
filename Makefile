.PHONY: up
aspireup:
	docker run --rm -it -d \
		-p 18888:18888 \
		-p 4317:18889 \
		--name aspire-dashboard \
		mcr.microsoft.com/dotnet/aspire-dashboard:latest
