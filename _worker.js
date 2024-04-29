/**
 * Welcome to Cloudflare Workers! This is your first worker.
 *
 * - Run "npm run dev" in your terminal to start a development server
 * - Open a browser tab at http://localhost:8787/ to see your worker in action
 * - Run "npm run deploy" to publish your worker
 *
 * Learn more at https://developers.cloudflare.com/workers/
 */

export default {
	async fetch(request, env, ctx) {
	  // 创建一个新的 URL 对象
	  let url = new URL(request.url);
	  let path = url.pathname.substring(1);
	  let isSecure = url.protocol.startsWith("https");
	  
	  if (path === "ws") {
		return handleWSRequest(request);
	  } else if (path === "locations") {
		let targetUrl = `http${isSecure ? 's' : ''}://speed.cloudflare.com/locations`;
		let cfRequest = new Request(targetUrl, request);
		let response = await fetch(cfRequest);
		return response;
	  } else {
		return handleSpeedTestRequest(request, path, isSecure);
	  }
	}
  };
  
  async function handleSpeedTestRequest(request, path, isSecure) {
	let bytes;
	if (!path) {
		// 路径为空，将 bytes 赋值为 100MB
		bytes = 100000000;
	} else {
	  // 其他路径，进行正常的处理
	  const regex = /^(\d+)([a-z]?)$/i;
	  const match = path.match(regex);
	  if (!match) {
		// 路径格式不正确，返回错误
		return new Response("路径格式不正确", {
		  status: 400,
		});
	  }
  
	  const bytesStr = match[1];
	  const unit = match[2].toLowerCase();
  
	  // 转换单位
	  bytes = parseInt(bytesStr, 10);
	  if (unit === "k") {
		bytes *= 1000;
	  } else if (unit === "m") {
		bytes *= 1000000;
	  } else if (unit === "g") {
		bytes *= 1000000000;
	  }
	}
  
	let targetUrl = `http${isSecure ? 's' : ''}://speed.cloudflare.com/__down?bytes=${bytes}`;
	  let cfRequest = new Request(targetUrl, request);
	  let response = await fetch(cfRequest);
  
	  // 将测试结果反馈给用户
	  return response;
  }
  
  async function handleWSRequest(request) {
	const upgradeHeader = request.headers.get('Upgrade');
	if (!upgradeHeader || upgradeHeader !== 'websocket') {
	  return new Response('Expected Upgrade: websocket', { status: 426 });
	}
  
	  /** @type {import("@cloudflare/workers-types").WebSocket[]} */
	  // @ts-ignore
	  const webSocketPair = new WebSocketPair();
	  const [client, webSocket] = Object.values(webSocketPair);
  
	  handleSession(webSocket);
  
	return new Response(null, {
	  status: 101,
	  // @ts-ignore
	  webSocket: client,
	});
  }
  
  async function handleSession(websocket) {
	websocket.accept()
	websocket.addEventListener("message", async message => {
	  console.log(message)
	})
  
	websocket.addEventListener("close", async evt => {
	  // Handle when a client closes the WebSocket connection
	  console.log(evt)
	})
  }