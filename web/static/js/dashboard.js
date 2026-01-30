function dashboard() {
  return {
    baseUrl: window.GO_CONFIG.BASE_URL,
    adminUser: "Admin",

    MAX_ASSET_KEYS: 7,
    systemError: false,

    stats: { total_items: 0, recent_uploads: [] },
    assets: [],

    // STATE MANAGEMENT
    loading: true, // Global Loading
    assetsLoading: false, // Assets Loading

    search: "",
    currentPath: "",
    folderViewEnabled: true,

    tab: "overview",
    viewMode: "gallery",
    sidebarOpen: false,
    currentFolder: [],

    currentPage: 1,
    currentPageInput: 1,
    itemsPerPage: 50,
    totalPages: 1,

    sortCol: "created_at",
    sortAsc: false,

    uploadModal: { open: false },
    deleteModal: { open: false, targetId: null, targetKey: "" },
    imageModal: {
      open: false,
      data: {},
      viewer: false,
      loading: false,
      tags: [],
      tagInput: "",
      newFile: null,
      previewUrl: null,
      showSettings: false,
      replaceMode: "square",
      replaceSize: 256,
      replaceScale: 100,
      viewerIndex: false,
    },
    toast: { show: false, message: "", type: "success" },
    blob: {
      secret: "",
      keys: "",
      file: null,
      loading: false,
      mode: "square",
      size: 256,
      scale: 100,
    },

    init() {
      if (!this.baseUrl.startsWith("http"))
        this.baseUrl = window.location.origin;
      else if (this.baseUrl.endsWith("/"))
        this.baseUrl = this.baseUrl.slice(0, -1);

      this.blob.secret = localStorage.getItem("octa_upload_secret") || "";
      const match = document.cookie.match(/auth_user=([^;]+)/);
      if (match) this.adminUser = match[1];

      // Restore State from Hash
      const hash = window.location.hash.substring(1);
      if (hash) {
        const parts = hash.split("/");
        if (parts[0]) this.tab = parts[0];
        if (parts[1]) this.viewMode = parts[1];
        if (parts[2] === "folder") {
          this.currentFolder = parts.slice(3).map(decodeURIComponent);
          this.folderViewEnabled = true;
        }
      }

      // First Load
      this.fetchInitialData();

      this.$watch("blob.secret", (val) =>
        localStorage.setItem("octa_upload_secret", val),
      );
      this.$watch("tab", () => this.updateHash());
      this.$watch("viewMode", () => this.updateHash());
      this.$watch("currentPath", () => this.updateHash());

      this.$watch("search", () => {
        this.currentPage = 1;
        this.fetchAssets();
      });
    },

    async apiCall(endpoint, options = {}) {
      const url = `${this.baseUrl}${endpoint}`;
      const headers = { ...options.headers };
      try {
        const res = await fetch(url, { ...options, headers });
        if (res.status === 401) {
          window.location.href = "/console/login";
          throw new Error("Session expired");
        }

        let data;
        const contentType = res.headers.get("content-type");
        if (contentType && contentType.indexOf("application/json") !== -1) {
          data = await res.json();
        } else {
          if (res.ok) return res;
          data = { message: res.statusText };
        }

        if (!res.ok) throw new Error(data.message || `Error ${res.status}`);
        return data;
      } catch (err) {
        if (err.message === "Failed to fetch")
          throw new Error("Cannot connect to server.");
        throw err;
      }
    },

    updateHash() {
      let hash = this.tab;
      if (this.tab === "library") {
        hash += "/" + this.viewMode;
        // If we are in a folder, add it to the URL
        if (this.currentPath && this.currentPath !== "") {
          const cleanPath = this.currentPath.replace(/\/$/, ""); //Remove the trailing slash (URL aesthetics)
          hash += "/folder/" + encodeURIComponent(cleanPath);
        }
      }
      window.location.hash = hash;
    },

    // getFolderCount: Calculates how many items appear in a folder
    getFolderCount(folderName) {
      // Target prefix: “path/foldername/”
      const targetPrefix = this.currentPath + folderName + "/";
      const uniqueAssetIds = new Set();

      this.assets.forEach((asset) => {
        const keys = asset.keys.split(",").map((k) => k.trim());
        // If any key starts with this folder path, count this asset
        if (keys.some((k) => k.startsWith(targetPrefix))) {
          uniqueAssetIds.add(asset.id);
        }
      });

      return uniqueAssetIds.size;
    },

    setTab(t) {
      this.tab = t;
      this.sidebarOpen = false;
    },
    setViewMode(m) {
      this.viewMode = m;
    },
    getPageTitle() {
      return this.tab === "overview"
        ? "Dashboard"
        : this.tab === "library"
          ? "Media Library"
          : "Settings";
    },
    getPageSubtitle() {
      return this.tab === "overview"
        ? "Overview & Statistics"
        : this.tab === "library"
          ? "Manage Assets"
          : "System Configuration";
    },

    navigateFolder(folderName) {
      this.currentPath += folderName + "/";
      this.search = "";
      this.currentPage = 1;
      this.currentPageInput = 1;
      this.fetchAssets();
    },

    goHome() {
      this.currentPath = "";
      this.search = "";

      this.currentPage = 1;
      this.currentPageInput = 1;
      this.fetchAssets();
      this.updateHash();
    },

    navigateToPathIndex(index) {
      const parts = this.pathParts;
      const newPath = parts.slice(0, index + 1).join("/") + "/";
      this.currentPath = newPath;
      this.search = "";
      this.currentPage = 1;
      this.currentPageInput = 1;
      this.fetchAssets();
      this.updateHash();
    },

    toggleFolderView() {
      this.folderViewEnabled = !this.folderViewEnabled;
    },

    navigateToBreadcrumb(index, folderName = "") {
      if (folderName != "") {
        this.search = folderName + "/";
      }

      this.currentFolder = this.currentFolder.slice(0, index + 1);
      this.search = "";

      this.fetchAssets();
    },

    get filteredAssets() {
      let result = this.assets;
      if (this.search) {
        const q = this.search.toLowerCase();
        result = result.filter(
          (a) =>
            a.keys.toLowerCase().includes(q) || a.id.toLowerCase().includes(q),
        );
      }
      return result.sort((a, b) => {
        let valA = a[this.sortCol];
        let valB = b[this.sortCol];
        if (this.sortCol === "size") {
          valA = Number(valA);
          valB = Number(valB);
        }
        if (valA < valB) return this.sortAsc ? -1 : 1;
        if (valA > valB) return this.sortAsc ? 1 : -1;
        return 0;
      });
    },

    // Virtual FileSystem Logic
    get currentViewItems() {
      const rawItems = this.filteredAssets;
      const prefix = this.currentPath; // Example: “” (Root) or “key/”

      // If folder view is closed or a search is being performed, return a flat list
      if (!this.folderViewEnabled || this.search) {
        return rawItems.map((a) => ({ type: "file", name: a.keys, ...a }));
      }

      const viewItems = [];
      const seenFolders = new Set();
      const seenFileIds = new Set();

      rawItems.forEach((asset) => {
        const keys = asset.keys
          .split(",")
          .map((k) => k.trim())
          .filter((k) => k);

        // If we are in the root directory and any key of this asset contains a folder,
        // this asset should not appear as a file in the root directory, it should only trigger a folder.
        const hasFolderKeys = keys.some((k) => k.includes("/"));

        keys.forEach((key) => {
          // Process the keys starting with our current path
          if (key.startsWith(prefix)) {
            const relativeKey = key.substring(prefix.length);

            if (relativeKey.includes("/")) {
              // There is a subfolder: “sub/image -> Take the “sub” part
              const folderName = relativeKey.split("/")[0];
              if (!seenFolders.has(folderName)) {
                seenFolders.add(folderName);
                viewItems.push({ type: "folder", name: folderName });
              }
            } else {
              // This asset is a file candidate at this level.
              // IF we are at the root and the asset has keys in other folders, do not display it at the root.
              if (prefix === "" && hasFolderKeys) {
                return;
              }

              if (!seenFileIds.has(asset.id)) {
                seenFileIds.add(asset.id);
                viewItems.push({ type: "file", name: relativeKey, ...asset });
              }
            }
          }
        });
      });

      return viewItems.sort((a, b) => {
        if (a.type === b.type) return a.name.localeCompare(b.name);
        return a.type === "folder" ? -1 : 1;
      });
    },

    get pathParts() {
      if (!this.currentPath) return [];
      return this.currentPath.split("/").filter((p) => p);
    },

    goToPage(p) {
      if (!p || isNaN(p)) p = 1;
      if (p < 1) p = 1;
      if (p > this.totalPages) p = this.totalPages;
      this.currentPage = p;
      this.currentPageInput = p;
      this.fetchAssets();
      // if(p>=1 && p<=this.totalPages) { this.currentPage = p; this.fetchAssets(); }
    },
    // nextPage() { if (this.currentPage < this.totalPages) this.goToPage(this.currentPage + 1); },
    //  prevPage() { if (this.currentPage > 1) this.goToPage(this.currentPage - 1); },

    nextPage() {
      this.goToPage(this.currentPage + 1);
    },
    prevPage() {
      this.goToPage(this.currentPage - 1);
    },

    // Fetchs
    async fetchInitialData() {
      this.loading = true;
      this.systemError = false;
      try {
        await Promise.all([this.fetchStats(), this.fetchAssets()]);
      } catch (e) {
        this.systemError = true;
      } finally {
        this.loading = false;
      }
    },

    // Fetch Stats
    async fetchStats() {
      try {
        this.stats = await this.apiCall("/console/api/stats");
      } catch (e) {
        console.error(e);
      }
    },

    formatUptime(seconds) {
      seconds = Math.max(0, Math.floor(seconds));

      const h = Math.floor(seconds / 3600);
      seconds %= 3600;
      const m = Math.floor(seconds / 60);
      const s = seconds % 60;

      const pad = (num) => num.toString().padStart(2, "0");

      if (h > 0) {
        return `${pad(h)}h ${pad(m)}m ${pad(s)}s`;
      }
      return `${pad(m)}m ${pad(s)}s`;
    },

    // Fetch Assets
    async fetchAssets() {
      this.assetsLoading = true;
      try {
        const params = new URLSearchParams({
          page: this.currentPage,
          limit: this.itemsPerPage,
        });
        //     if (this.search) params.append("q", this.search);

        if (this.search) {
          // Global Search
          params.append("q", this.search);
        } else if (this.currentPath) {
          // Folder Internal (Prefix Search)
          params.append("q", this.currentPath);
        }

        const assetsData = await this.apiCall(
          `/console/api/assets?${params.toString()}`,
        );

        this.assets = assetsData.items || [];
        this.totalPages = assetsData.total_pages || 1;
        this.currentPageInput = this.currentPage;

        if (assetsData.total_items)
          this.stats.total_items = assetsData.total_items;
      } catch (e) {
        this.showToast("Failed to load assets", "error");
      } finally {
        this.assetsLoading = false;
      }
    },

    handleFile(e) {
      this.blob.file = e.target.files[0];
    },

    async uploadBlob() {
      if (!this.blob.file || !this.blob.keys || !this.blob.secret) return;

      let finalKey = this.blob.keys || this.blob.file.name;
      // If we are in a folder, add the prefix
      if (this.currentPath && !finalKey.startsWith(this.currentPath)) {
        finalKey = this.currentPath + finalKey;
      }

      const keyCount = this.blob.keys.split(",").filter((k) => k.trim()).length;
      if (keyCount > this.MAX_ASSET_KEYS) {
        this.showToast(`Key limit exceeded.`, "error");
        return;
      }

      this.blob.loading = true;
      const fd = new FormData();
      // this.blob.keys
      fd.append("keys", finalKey);
      fd.append("avatar", this.blob.file);
      fd.append("mode", this.blob.mode);
      if (this.blob.mode !== "original") fd.append("size", this.blob.size);
      if (this.blob.mode === "scale") fd.append("scale", this.blob.scale);

      try {
        const res = await fetch(`${this.baseUrl}/upload`, {
          method: "POST",
          headers: { "X-Secret-Key": this.blob.secret },
          body: fd,
        });
        if (!res.ok) {
          const errData = await res.json();
          throw new Error(errData.message);
        }
        this.showToast("Asset Uploaded Successfully");
        this.blob.file = null;
        this.blob.keys = "";
        this.uploadModal.open = false;

        // Update both the list and the statistics after upload
        this.fetchStats();
        this.fetchAssets();
      } catch (e) {
        this.showToast(e.message, "error");
      } finally {
        this.blob.loading = false;
      }
    },

    openImageModal(item, openViewer = false) {
      this.imageModal.data = item;
      this.imageModal.tags = item.keys
        ? item.keys
            .split(",")
            .map((k) => k.trim())
            .filter((k) => k)
        : [];
      this.imageModal.tagInput = "";
      this.clearReplace();
      this.imageModal.open = !openViewer;
      this.imageModal.viewer = openViewer;
    },
    openViewerFromDetail() {
      this.imageModal.viewer = true;
      this.imageModal.viewerIndex = true;
    },
    closeViewerImageModal() {
      this.imageModal.viewer = false;
      this.imageModal.viewerIndex = false;
    },
    addTag() {
      const val = this.imageModal.tagInput.trim();
      if (!val) return;
      if (this.imageModal.tags.length >= this.MAX_ASSET_KEYS) {
        this.showToast(`Max keys allowed.`, "info");
        return;
      }
      if (!this.imageModal.tags.includes(val)) this.imageModal.tags.push(val);
      this.imageModal.tagInput = "";
    },
    removeTag(index) {
      this.imageModal.tags.splice(index, 1);
    },
    handleReplaceFile(e) {
      const file = e.target.files[0];
      if (file) {
        this.imageModal.newFile = file;
        this.imageModal.previewUrl = URL.createObjectURL(file);
        this.imageModal.showSettings = true;
      }
    },
    clearReplace() {
      this.imageModal.newFile = null;
      this.imageModal.previewUrl = null;
      this.imageModal.showSettings = false;
      if (this.$refs.replaceInput) this.$refs.replaceInput.value = "";
    },
    async saveChanges() {
      this.imageModal.loading = true;
      const keysStr = this.imageModal.tags.join(",");
      try {
        if (this.imageModal.newFile) {
          const fd = new FormData();
          fd.append("keys", keysStr);
          fd.append("avatar", this.imageModal.newFile);
          fd.append("mode", this.imageModal.replaceMode);
          if (this.imageModal.replaceMode !== "original")
            fd.append("size", this.imageModal.replaceSize);
          if (this.imageModal.replaceMode === "scale")
            fd.append("scale", this.imageModal.replaceScale);
          const res = await fetch(`${this.baseUrl}/upload`, {
            method: "POST",
            headers: { "X-Secret-Key": this.blob.secret },
            body: fd,
          });
          if (!res.ok) throw new Error("Replace failed");
          this.showToast("Image Replaced");
        } else {
          await this.apiCall(`/console/api/assets/${this.imageModal.data.id}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ keys: keysStr }),
          });
          this.showToast("Keys updated");
        }
        // Refresh data after update
        this.fetchStats();
        this.fetchAssets();
        this.imageModal.open = false;
      } catch (e) {
        this.showToast(e.message, "error");
      } finally {
        this.imageModal.loading = false;
      }
    },
    confirmDelete(item) {
      this.deleteModal.targetId = item.id;
      this.deleteModal.targetKey = item.keys;
      this.deleteModal.open = true;
    },
    async executeDelete() {
      try {
        await this.apiCall(`/console/api/assets/${this.deleteModal.targetId}`, {
          method: "DELETE",
        });

        // Post-deletion update
        this.stats.total_count--;
        this.fetchAssets();
        this.fetchStats();

        this.showToast("Deleted");
        this.deleteModal.open = false;
        this.imageModal.open = false;
      } catch (e) {
        this.showToast(e.message, "error");
      }
    },
    async downloadBackup() {
      const secret = this.blob.secret;
      if (!secret) return this.showToast("Enter Secret Key first!", "error");
      window.location.href = `${this.baseUrl}/console/api/backup?secret=${secret}`;
    },
    async logout() {
      try {
        await this.apiCall(`/console/api/logout`, { method: "POST" });
      } catch (e) {}
      window.location.href = "/console/login";
    },
    formatBytes(bytes, decimals = 2) {
      if (!+bytes) return "0 B";
      const k = 1024;
      const dm = decimals < 0 ? 0 : decimals;
      const sizes = ["B", "KB", "MB", "GB", "TB"];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
    },
    formatSmartDate(dateString) {
      if (!dateString) return "-";
      const date = new Date(dateString);
      return date.toLocaleDateString();
    },
    showToast(msg, type = "success") {
      this.toast.message = msg;
      this.toast.type = type;
      this.toast.show = true;
      setTimeout(() => (this.toast.show = false), 3000);
    },
    async copyText(txt) {
      try {
        await navigator.clipboard.writeText(txt);
        this.showToast("Copied");
      } catch (e) {}
    },
  };
}


window.dashboard = dashboard;