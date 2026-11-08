"use strict";

const $ = (id) => document.getElementById(id);
const state = { members: [], loaded: false, editing: null, removing: null, busy: false };
const memberDialog = $("member-dialog");
const deleteDialog = $("delete-dialog");

async function api(path, options = {}) {
  let response;
  try {
    response = await fetch(new URL(`api${path}`, document.baseURI), {
      ...options,
      headers: { "Content-Type": "application/json", ...options.headers },
      signal: AbortSignal.timeout(15000),
    });
  } catch {
    throw new Error("The API could not be reached. Check your connection and try again.");
  }
  if (response.status === 204) return null;
  const body = await response.json().catch(() => null);
  if (!response.ok) throw new Error(body?.error || `Request failed (${response.status}). Please try again.`);
  if (!body) throw new Error("The API returned an unexpected response.");
  return body;
}

function notice(message, error = false) {
  $("notice").textContent = message;
  $("notice").classList.toggle("error", error);
  $("notice").hidden = !message;
}

function element(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function renderMembers() {
  const query = $("search").value.trim().toLowerCase();
  const visible = state.members.filter((member) =>
    [member.name, member.email, member.role].some((value) => value.toLowerCase().includes(query))
  );
  $("member-list").replaceChildren();
  $("directory-count").textContent = state.members.length;
  $("total-count").textContent = state.loaded ? state.members.length : "—";
  $("admin-count").textContent = state.loaded ? state.members.filter((member) => member.role.toLowerCase() === "admin").length : "—";

  for (const member of visible) {
    const row = document.createElement("tr");
    const identityCell = document.createElement("td");
    const identity = element("div", "member-cell");
    const initials = member.name.trim().split(/\s+/).slice(0, 2).map((part) => Array.from(part)[0]).join("").toUpperCase();
    const avatar = element("span", "avatar", initials);
    avatar.setAttribute("aria-hidden", "true");
    const details = document.createElement("div");
    details.append(element("span", "member-name", member.name), element("span", "member-id", `Member #${member.id}`));
    identity.append(avatar, details);
    identityCell.append(identity);
    const roleCell = document.createElement("td");
    roleCell.append(element("span", `role${member.role.toLowerCase() === "admin" ? " admin" : ""}`, member.role));
    const actions = element("td", "actions-cell");
    const edit = element("button", "text-button", "Edit");
    edit.type = "button";
    edit.setAttribute("aria-label", `Edit ${member.name}`);
    edit.addEventListener("click", () => openMember(member));
    const remove = element("button", "text-button remove", "Remove");
    remove.type = "button";
    remove.setAttribute("aria-label", `Remove ${member.name}`);
    remove.addEventListener("click", () => openDelete(member));
    actions.append(edit, remove);
    row.append(identityCell, element("td", "email-cell", member.email), roleCell, actions);
    $("member-list").append(row);
  }

  $("empty-state").hidden = visible.length > 0 || !state.loaded;
  $("empty-title").textContent = query ? "No matching members" : "No members yet";
  $("empty-description").textContent = query ? "Try another name, email address, or role." : "Add your first member to get started.";
  $("results-count").textContent = state.loaded ? `Showing ${visible.length} of ${state.members.length} members` : "No data loaded";
}

async function loadMembers() {
  $("refresh").disabled = true;
  $("loading").hidden = state.loaded;
  try {
    const body = await api("/members");
    if (!Array.isArray(body.data)) throw new Error("The API returned an unexpected member list.");
    state.members = body.data;
    state.loaded = true;
    $("connection-status").textContent = "Connected";
    renderMembers();
    return true;
  } catch (error) {
    $("connection-status").textContent = "Unavailable";
    notice(`${error.message}${state.loaded ? " Showing the last loaded list." : ""}`, true);
    renderMembers();
    return false;
  } finally {
    $("refresh").disabled = false;
    $("loading").hidden = true;
  }
}

function openMember(member = null) {
  state.editing = member;
  $("member-form").reset();
  $("member-name").value = member?.name || "";
  $("member-email").value = member?.email || "";
  $("member-role").value = member?.role || "member";
  $("form-title").textContent = member ? "Edit member" : "Add a member";
  $("save-member").textContent = member ? "Save changes" : "Add member";
  $("form-error").hidden = true;
  memberDialog.showModal();
  $("member-name").focus();
}

function openDelete(member) {
  state.removing = member;
  $("delete-description").textContent = `${member.name} will be removed from the workspace. This cannot be undone.`;
  $("delete-error").hidden = true;
  deleteDialog.showModal();
  $("cancel-delete").focus();
}

function lockDialog(dialog, locked) {
  state.busy = locked;
  dialog.querySelectorAll("button, input").forEach((control) => { control.disabled = locked; });
}

$("member-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  if (state.busy) return;
  const editing = state.editing;
  const input = {
    name: $("member-name").value.trim(),
    email: $("member-email").value.trim(),
    role: $("member-role").value.trim() || "member",
  };
  $("form-error").hidden = true;
  lockDialog(memberDialog, true);
  $("save-member").textContent = "Saving…";
  try {
    await api(editing ? `/members/${editing.id}` : "/members", { method: editing ? "PUT" : "POST", body: JSON.stringify(input) });
    memberDialog.close();
    notice(editing ? "Member details updated." : `${input.name} added to the workspace.`);
    await loadMembers();
  } catch (error) {
    $("form-error").textContent = error.message;
    $("form-error").hidden = false;
  } finally {
    lockDialog(memberDialog, false);
    $("save-member").textContent = editing ? "Save changes" : "Add member";
  }
});

$("delete-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  if (state.busy || !state.removing) return;
  $("delete-error").hidden = true;
  lockDialog(deleteDialog, true);
  $("confirm-delete").textContent = "Removing…";
  try {
    await api(`/members/${state.removing.id}`, { method: "DELETE" });
    deleteDialog.close();
    notice(`${state.removing.name} removed from the workspace.`);
    await loadMembers();
  } catch (error) {
    $("delete-error").textContent = error.message;
    $("delete-error").hidden = false;
  } finally {
    lockDialog(deleteDialog, false);
    $("confirm-delete").textContent = "Remove member";
  }
});

$("add-member").addEventListener("click", () => openMember());
$("close-dialog").addEventListener("click", () => memberDialog.close());
$("cancel-dialog").addEventListener("click", () => memberDialog.close());
$("cancel-delete").addEventListener("click", () => deleteDialog.close());
$("search").addEventListener("input", renderMembers);
$("refresh").addEventListener("click", () => { notice(""); loadMembers(); });
for (const dialog of [memberDialog, deleteDialog]) {
  dialog.addEventListener("cancel", (event) => { if (state.busy) event.preventDefault(); });
}
loadMembers();
